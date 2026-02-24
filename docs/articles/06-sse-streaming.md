# Server-Sent Events in Go: Building Real-Time AI Streaming Without WebSockets

*Part 6 of the "Building Production-Ready AI Agent APIs in Go" series*

---

Every AI chat interface you have used — ChatGPT, Claude, Gemini — shows you text as it is generated, word by word. The LLM does not wait until the full response is complete before sending it. It streams.

Implementing this in Go involves four layers working together: the LLM provider sends SSE (Server-Sent Events) → we parse it into a typed Go channel → the use case sends events downstream → the HTTP handler writes them to the client with `http.Flusher`.

The good news: Go's standard library handles all of this. No WebSockets, no third-party streaming library, no special runtime. Just `bufio.Reader`, channels, and `http.Flusher`.

---

## Why SSE Over WebSockets for LLM Streaming

WebSockets provide full-duplex communication — both client and server can send messages at any time. For LLM streaming, that capability is unnecessary. The request flow is:

```
Client → sends full message (request)
Server → streams tokens back (response)
```

This is still fundamentally request-response. The server streams the response, but the client is not sending data during streaming. SSE is designed for exactly this pattern: the server pushes data to the client over a persistent HTTP connection. The client reads it as a stream.

SSE advantages over WebSockets for this use case:
- Works through HTTP/2 proxies and load balancers without special configuration
- Automatic reconnection built into browser clients
- Uses standard HTTP headers — no protocol upgrade negotiation
- Simpler to implement and debug (it is just text over HTTP)
- Compatible with existing HTTP middleware (auth, rate limiting, logging all work unchanged)

---

## The LLMProvider Interface: Streaming as a Contract

The streaming contract is defined in `internal/application/port/llm_provider.go`:

```go
type StreamEvent struct {
    Type     string             `json:"type"`
    Delta    string             `json:"delta,omitempty"`   // Text chunk
    ToolCall *toolspec.ToolCall `json:"tool_call,omitempty"`
    Usage    *UsageInfo         `json:"usage,omitempty"`
    Error    error              `json:"-"`   // Not serialized to JSON
    Done     bool               `json:"done,omitempty"`
}

type LLMProvider interface {
    Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
    ChatStream(ctx context.Context, req ChatRequest) (<-chan StreamEvent, error)
}
```

`ChatStream` returns `<-chan StreamEvent` — a receive-only channel. The caller reads events from this channel until `event.Done == true` or `event.Error != nil`. The channel is closed by the implementor when the stream ends.

`StreamEvent` has a `Type` field with these possible values:
- `"content"` — a text chunk, `Delta` contains the token(s)
- `"tool_call"` — the LLM is requesting a tool call, `ToolCall` is populated
- `"finish"` — the stream has ended, `Delta` is the finish reason, `Usage` has token counts
- Plus `Done: true` and `Error: err` for terminal states

The `Error` field has `json:"-"` — it is never serialized to JSON. When we forward a `StreamEvent` to the client, errors go through a separate error event rather than being exposed as an internal Go error.

---

## The LiteLLM Client: Parsing SSE

The implementation is in `internal/infrastructure/llm/litellm/client.go`. `ChatStream` makes the HTTP request and starts a goroutine:

```go
func (c *Client) ChatStream(ctx context.Context, req port.ChatRequest) (<-chan port.StreamEvent, error) {
    req.Stream = true   // Tell the LLM to stream

    body, _ := json.Marshal(req)
    httpReq, _ := http.NewRequestWithContext(ctx, "POST",
        c.baseURL+"/chat/completions", bytes.NewReader(body))

    httpReq.Header.Set("Content-Type", "application/json")
    httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
    httpReq.Header.Set("Accept", "text/event-stream")

    resp, err := c.httpClient.Do(httpReq)
    if err != nil {
        return nil, fmt.Errorf("do request: %w", err)
    }

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        resp.Body.Close()
        return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
    }

    events := make(chan port.StreamEvent)
    go c.readSSE(ctx, resp, events)  // Parse SSE in a goroutine

    return events, nil
}
```

Notice: we check the HTTP status before starting the goroutine. If the request fails (401 Unauthorized, 400 Bad Request, etc.), we return an error synchronously. The goroutine only starts if the connection was established successfully.

The SSE parser:

```go
func (c *Client) readSSE(ctx context.Context, resp *http.Response, events chan<- port.StreamEvent) {
    defer close(events)        // Always close the channel when done
    defer resp.Body.Close()    // Always close the response body

    reader := bufio.NewReader(resp.Body)

    for {
        // Check for context cancellation (client disconnect)
        select {
        case <-ctx.Done():
            events <- port.StreamEvent{Error: ctx.Err()}
            return
        default:
        }

        line, err := reader.ReadString('\n')
        if err != nil {
            if err != io.EOF {
                events <- port.StreamEvent{Error: err}
            }
            events <- port.StreamEvent{Done: true}
            return
        }

        line = strings.TrimSpace(line)

        if line == "" {
            continue  // Empty lines are SSE event separators — skip
        }

        if !strings.HasPrefix(line, "data: ") {
            continue  // Ignore SSE comment lines (": keep-alive") and headers
        }

        data := strings.TrimPrefix(line, "data: ")

        if data == "[DONE]" {
            events <- port.StreamEvent{Done: true}
            return
        }

        // Parse the JSON chunk
        var chunk struct {
            Choices []struct {
                Delta struct {
                    Content   string              `json:"content"`
                    ToolCalls []toolspec.ToolCall `json:"tool_calls"`
                } `json:"delta"`
                FinishReason string `json:"finish_reason"`
            } `json:"choices"`
            Usage *port.UsageInfo `json:"usage"`
        }

        if err := json.Unmarshal([]byte(data), &chunk); err != nil {
            continue  // Skip malformed chunks (can happen with some providers)
        }

        if len(chunk.Choices) > 0 {
            choice := chunk.Choices[0]

            if choice.Delta.Content != "" {
                events <- port.StreamEvent{Type: "content", Delta: choice.Delta.Content}
            }

            for _, tc := range choice.Delta.ToolCalls {
                events <- port.StreamEvent{Type: "tool_call", ToolCall: &tc}
            }

            if choice.FinishReason != "" {
                events <- port.StreamEvent{Type: "finish", Delta: choice.FinishReason, Usage: chunk.Usage}
            }
        }
    }
}
```

A few implementation details worth noting:

**`bufio.NewReader(resp.Body)`** — The standard `bufio.NewReader` wraps the HTTP response body with buffering. `ReadString('\n')` reads until a newline, which is how SSE lines are delimited.

**The `[DONE]` sentinel** — OpenAI's SSE format uses `data: [DONE]` to signal stream completion. This is not JSON — it is a literal string. We check for it explicitly before attempting JSON parsing.

**The `ctx.Done()` select** — Between each line, we check if the context is cancelled. This is how client disconnects propagate. When the HTTP client closes its connection, the request context is cancelled. The select fires, we send an error event, and return — which closes the channel and the response body. No goroutine leak.

**Silently skip malformed JSON** — `json.Unmarshal` failures are ignored with `continue`. Some SSE providers send occasional keep-alive messages or other non-JSON data. Silently skipping them is more resilient than returning an error.

---

## The SSE Response Helpers

The `internal/pkg/response/response.go` package provides helpers for the HTTP handler side:

```go
// Stream sets the SSE headers on the response
func Stream(w http.ResponseWriter) {
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.Header().Set("X-Accel-Buffering", "no")   // Disable nginx buffering
}

// SSEEvent sends a named SSE event
func SSEEvent(w http.ResponseWriter, event, data string) {
    if event != "" {
        fmt.Fprintf(w, "event: %s\n", event)
    }
    fmt.Fprintf(w, "data: %s\n\n", data)  // Double newline = event separator
    if f, ok := w.(http.Flusher); ok {
        f.Flush()
    }
}

// SSEData sends a JSON-serialized data event
func SSEData(w http.ResponseWriter, data any) {
    b, _ := json.Marshal(data)
    fmt.Fprintf(w, "data: %s\n\n", string(b))
    if f, ok := w.(http.Flusher); ok {
        f.Flush()
    }
}
```

**`X-Accel-Buffering: no`** is the nginx-specific header that tells the reverse proxy not to buffer the response. Without this, nginx waits for the full response before forwarding it — defeating the purpose of streaming. Other proxies have equivalent headers.

**`http.Flusher`** — Go's `http.ResponseWriter` optionally implements `http.Flusher` with a single method: `Flush()`. This forces buffered data to be written to the client immediately. Without calling `Flush()` after each `fmt.Fprintf`, Go's HTTP layer might batch multiple writes and send them together — also defeating the purpose of streaming.

The `ok` check (`if f, ok := w.(http.Flusher); ok`) is defensive: not all `ResponseWriter` implementations support flushing (test recorders, for example). In production, `net/http`'s built-in `ResponseWriter` always implements `Flusher`.

---

## The Chat Handler: Putting It Together

The complete streaming handler in `internal/infrastructure/http/handler/chat_handler.go`:

```go
func (h *ChatHandler) ChatCompletions(w http.ResponseWriter, r *http.Request) {
    user := middleware.GetUserFromContext(r.Context())
    token := middleware.GetTokenFromContext(r.Context())

    var req request.ChatCompletionRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
        return
    }

    // Branch: streaming or non-streaming
    if req.Stream {
        h.handleStreamingChat(w, r, user, token, req)
        return
    }

    // Non-streaming: wait for full response
    result, err := h.sendMessage.Execute(r.Context(), chat.SendMessageInput{
        UserID: user.ID, Messages: req.Messages, Model: req.Model,
    })
    if err != nil {
        response.Error(w, http.StatusInternalServerError, "CHAT_ERROR", err.Error())
        return
    }
    response.JSON(w, http.StatusOK, result)
}

func (h *ChatHandler) handleStreamingChat(
    w http.ResponseWriter,
    r *http.Request,
    user *entity.User,
    _ *entity.Token,
    req request.ChatCompletionRequest,
) {
    response.Stream(w)  // Set SSE headers

    events, err := h.sendMessage.ExecuteStream(r.Context(), chat.SendMessageInput{
        UserID: user.ID, Messages: req.Messages, Model: req.Model,
    })
    if err != nil {
        response.SSEData(w, map[string]string{"error": err.Error()})
        return
    }

    for event := range events {
        if event.Error != nil {
            response.SSEData(w, map[string]string{"error": event.Error.Error()})
            break
        }
        response.SSEData(w, event)  // Forward each event to the client
        if event.Done {
            break
        }
    }

    response.SSEEvent(w, "", "[DONE]")  // OpenAI-compatible termination
}
```

The streaming handler:
1. Sets SSE headers
2. Gets an event channel from the use case
3. Range-loops over the channel, writing each event
4. Breaks on error or `Done`
5. Sends `[DONE]` as the final event

The `for event := range events` loop is elegant — it naturally terminates when the channel is closed. The goroutine inside the LiteLLM client closes the channel when it is done, which unblocks the range loop.

---

## Context Cancellation: The Full Pipeline

What happens when the client disconnects mid-stream?

Go's `net/http` server detects client disconnection and cancels the request's `context.Context`. Here is how that propagates through our stack:

```
Client disconnects
  → net/http cancels r.Context()
  → handleStreamingChat is using r.Context() via the range loop
  → r.Context() is passed to sendMessage.ExecuteStream()
  → ExecuteStream passes ctx to litellm.ChatStream()
  → ChatStream creates httpReq with http.NewRequestWithContext(ctx, ...)
  → The HTTP request to LiteLLM is also cancelled
  → readSSE's select{case <-ctx.Done():} fires
  → readSSE sends an error event and returns
  → events channel is closed
  → for event := range events loop in handleStreamingChat terminates
```

The entire pipeline cleans up through a single context cancellation. No goroutine leaks, no open HTTP connections to LiteLLM, no wasted tokens being generated for a disconnected client.

This is why `context.Context` is so important in Go. It is not bureaucracy — it is the mechanism that lets you thread cancellation signals through an entire call stack.

---

## Testing Streaming Handlers

Testing SSE handlers requires a custom response recorder that implements `http.Flusher`:

```go
type flushableRecorder struct {
    *httptest.ResponseRecorder
    flushed int
}

func (f *flushableRecorder) Flush() {
    f.flushed++
}

func TestChatHandler_Streaming(t *testing.T) {
    // Create a mock LLM that returns streaming events
    mockLLM := &MockLLMProvider{}
    events := make(chan port.StreamEvent, 3)
    events <- port.StreamEvent{Type: "content", Delta: "Hello"}
    events <- port.StreamEvent{Type: "content", Delta: " world"}
    events <- port.StreamEvent{Done: true}
    close(events)
    mockLLM.On("ChatStream", mock.Anything, mock.Anything).Return(events, nil)

    // Setup handler and recorder
    handler := NewChatHandler(/* ... */)
    rec := &flushableRecorder{ResponseRecorder: httptest.NewRecorder()}

    req := httptest.NewRequest("POST", "/v1/chat/completions",
        strings.NewReader(`{"stream": true, "messages": [{"role":"user","content":"hi"}]}`))

    handler.ChatCompletions(rec, req)

    // Verify SSE headers were set
    assert.Equal(t, "text/event-stream", rec.Header().Get("Content-Type"))

    // Verify events were written and flushed
    assert.Contains(t, rec.Body.String(), `"delta":"Hello"`)
    assert.Contains(t, rec.Body.String(), "[DONE]")
    assert.Greater(t, rec.flushed, 0)
}
```

The `flushableRecorder` wraps `httptest.ResponseRecorder` and adds a no-op `Flush()` method. This satisfies the `http.Flusher` interface check in `SSEData()` without actually flushing anything (since it is a test recorder, not a real connection).

---

## What We Just Learned

- SSE is simpler than WebSockets for LLM streaming — it is just HTTP with persistent connections and text events
- `LLMProvider.ChatStream()` returns `<-chan StreamEvent` — a typed, closeable channel
- `bufio.NewReader` + `ReadString('\n')` parses SSE line by line
- Context cancellation propagates from client disconnect → request context → HTTP request → goroutine shutdown
- `http.Flusher.Flush()` must be called after each write — otherwise Go's HTTP layer buffers events
- `X-Accel-Buffering: no` disables nginx proxy buffering
- The `[DONE]` sentinel is checked as a literal string before JSON parsing
- Testing requires a custom recorder that implements `http.Flusher`

---

*Next in the series: [Article 7 — Human-in-the-Loop AI: Pause, Approve, and Resume Agentic Workflows](#)*

*Previous: [Article 5 — JWT Meets API Keys: Dual Auth in Go](#)*
