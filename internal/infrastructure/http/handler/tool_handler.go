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

// ListTools godoc
// @Summary      List available tools
// @Description  Returns the list of tools the authenticated token is allowed to use
// @Tags         tools
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.SwaggerToolsResponse
// @Failure      401  {object}  response.SwaggerErrorResponse
// @Router       /v1/tools [get]
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

// ExecuteTool godoc
// @Summary      Execute a tool
// @Description  Executes a single tool call and returns the result
// @Tags         tools
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      toolspec.ToolCall                  true  "Tool call"
// @Success      200   {object}  response.SwaggerToolResultResponse
// @Failure      400   {object}  response.SwaggerErrorResponse
// @Failure      401   {object}  response.SwaggerErrorResponse
// @Failure      500   {object}  response.SwaggerErrorResponse
// @Router       /v1/tools/execute [post]
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

// ExecuteToolBatch godoc
// @Summary      Execute tools in parallel
// @Description  Executes multiple tool calls concurrently and returns all results
// @Tags         tools
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      []toolspec.ToolCall                 true  "Array of tool calls"
// @Success      200   {object}  response.SwaggerBatchResultResponse
// @Failure      400   {object}  response.SwaggerErrorResponse
// @Failure      401   {object}  response.SwaggerErrorResponse
// @Failure      500   {object}  response.SwaggerErrorResponse
// @Router       /v1/tools/batch [post]
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
