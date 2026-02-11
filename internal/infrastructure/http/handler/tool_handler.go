package handler

import (
	"encoding/json"
	"net/http"

	"github.com/wyuneed/go-agent-api/internal/application/usecase/tool"
	"github.com/wyuneed/go-agent-api/internal/infrastructure/http/middleware"
	"github.com/wyuneed/go-agent-api/internal/pkg/response"
	"github.com/wyuneed/go-agent-api/pkg/toolspec"
)

type ToolHandler struct {
	registry    *tool.ToolRegistry
	executeTool *tool.ExecuteToolUseCase
}

func NewToolHandler(registry *tool.ToolRegistry, executeTool *tool.ExecuteToolUseCase) *ToolHandler {
	return &ToolHandler{
		registry:    registry,
		executeTool: executeTool,
	}
}

// GET /v1/tools
func (h *ToolHandler) ListTools(w http.ResponseWriter, r *http.Request) {
	token := middleware.GetTokenFromContext(r.Context())

	var tools []toolspec.Tool
	if token != nil {
		tools = h.registry.ListForToken(token.AllowedTools)
	} else {
		tools = h.registry.List()
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"tools": tools,
	})
}

// POST /v1/tools/execute
func (h *ToolHandler) ExecuteTool(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	token := middleware.GetTokenFromContext(r.Context())

	var req toolspec.ToolCall
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid tool call format")
		return
	}

	result, err := h.executeTool.Execute(r.Context(), tool.ExecuteToolInput{
		ToolCall: req,
		User:     user,
		Token:    token,
	})
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "TOOL_ERROR", err.Error())
		return
	}

	response.JSON(w, http.StatusOK, result.Result)
}

// POST /v1/tools/batch
func (h *ToolHandler) ExecuteToolBatch(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	token := middleware.GetTokenFromContext(r.Context())

	var calls []toolspec.ToolCall
	if err := json.NewDecoder(r.Body).Decode(&calls); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid tool calls format")
		return
	}

	results := make([]any, len(calls))
	errChan := make(chan error, len(calls))

	for i, tc := range calls {
		go func(idx int, call toolspec.ToolCall) {
			result, err := h.executeTool.Execute(r.Context(), tool.ExecuteToolInput{
				ToolCall: call,
				User:     user,
				Token:    token,
			})
			if err != nil {
				errChan <- err
				return
			}
			results[idx] = result.Result
			errChan <- nil
		}(i, tc)
	}

	for range calls {
		<-errChan
	}

	response.JSON(w, http.StatusOK, map[string]any{"results": results})
}
