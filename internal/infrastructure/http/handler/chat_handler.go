package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/wyuneed/go-agent-api/internal/application/dto/request"
	"github.com/wyuneed/go-agent-api/internal/application/usecase/chat"
	"github.com/wyuneed/go-agent-api/internal/domain/entity"
	"github.com/wyuneed/go-agent-api/internal/infrastructure/http/middleware"
	"github.com/wyuneed/go-agent-api/internal/pkg/response"
)

type ChatHandler struct {
	sendMessage       *chat.SendMessageUseCase
	getConversation   *chat.GetConversationUseCase
	listConversations *chat.ListConversationsUseCase
	approveAction     *chat.ApproveActionUseCase
}

func NewChatHandler(
	sendMessage *chat.SendMessageUseCase,
	getConversation *chat.GetConversationUseCase,
	listConversations *chat.ListConversationsUseCase,
	approveAction *chat.ApproveActionUseCase,
) *ChatHandler {
	return &ChatHandler{
		sendMessage:       sendMessage,
		getConversation:   getConversation,
		listConversations: listConversations,
		approveAction:     approveAction,
	}
}

// ChatCompletions godoc
// @Summary      Chat completions (OpenAI-compatible)
// @Description  Sends messages to the AI agent and returns a completion. Set stream=true to receive Server-Sent Events.
// @Tags         chat
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      request.ChatCompletionRequest         true  "Chat completion request"
// @Success      200   {object}  response.SwaggerChatCompletionResponse
// @Failure      400   {object}  response.SwaggerErrorResponse
// @Failure      401   {object}  response.SwaggerErrorResponse
// @Failure      500   {object}  response.SwaggerErrorResponse
// @Router       /v1/chat/completions [post]
func (h *ChatHandler) ChatCompletions(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	token := middleware.GetTokenFromContext(r.Context())

	var req request.ChatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	if req.Stream {
		h.handleStreamingChat(w, r, user, token, req)
		return
	}

	result, err := h.sendMessage.Execute(r.Context(), chat.SendMessageInput{
		UserID:   user.ID,
		Messages: req.Messages,
		Model:    req.Model,
	})
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "CHAT_ERROR", err.Error())
		return
	}

	response.JSON(w, http.StatusOK, result)
}

func (h *ChatHandler) handleStreamingChat(w http.ResponseWriter, r *http.Request, user *entity.User, _ *entity.Token, req request.ChatCompletionRequest) {
	response.Stream(w)

	events, err := h.sendMessage.ExecuteStream(r.Context(), chat.SendMessageInput{
		UserID:   user.ID,
		Messages: req.Messages,
		Model:    req.Model,
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
		response.SSEData(w, event)
		if event.Done {
			break
		}
	}

	response.SSEEvent(w, "", "[DONE]")
}

// CreateConversation godoc
// @Summary      Create a conversation
// @Description  Creates a new conversation and sends the first message to the agent
// @Tags         conversations
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      request.CreateConversationRequest     true  "Conversation details"
// @Success      201   {object}  response.SwaggerSendMessageResponse
// @Failure      400   {object}  response.SwaggerErrorResponse
// @Failure      401   {object}  response.SwaggerErrorResponse
// @Failure      500   {object}  response.SwaggerErrorResponse
// @Router       /v1/conversations [post]
func (h *ChatHandler) CreateConversation(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())

	var req request.CreateConversationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	result, err := h.sendMessage.Execute(r.Context(), chat.SendMessageInput{
		UserID:    user.ID,
		Content:   req.Message,
		AgentType: req.AgentType,
	})
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "CHAT_ERROR", err.Error())
		return
	}

	response.JSON(w, http.StatusCreated, result)
}

// SendMessage godoc
// @Summary      Send a message
// @Description  Sends a message to an existing conversation and returns the agent's reply
// @Tags         conversations
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                                true  "Conversation ID (UUID)"
// @Param        body  body      request.SendMessageRequest            true  "Message content"
// @Success      200   {object}  response.SwaggerSendMessageResponse
// @Failure      400   {object}  response.SwaggerErrorResponse
// @Failure      401   {object}  response.SwaggerErrorResponse
// @Failure      500   {object}  response.SwaggerErrorResponse
// @Router       /v1/conversations/{id}/messages [post]
func (h *ChatHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())

	convID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid conversation ID")
		return
	}

	var req request.SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	result, err := h.sendMessage.Execute(r.Context(), chat.SendMessageInput{
		ConversationID: &convID,
		UserID:         user.ID,
		Content:        req.Message,
	})
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "CHAT_ERROR", err.Error())
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// GetConversation godoc
// @Summary      Get a conversation
// @Description  Returns a conversation with its full message history
// @Tags         conversations
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string                                true  "Conversation ID (UUID)"
// @Success      200  {object}  response.SwaggerConversationResponse
// @Failure      400  {object}  response.SwaggerErrorResponse
// @Failure      401  {object}  response.SwaggerErrorResponse
// @Failure      404  {object}  response.SwaggerErrorResponse
// @Router       /v1/conversations/{id} [get]
func (h *ChatHandler) GetConversation(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())

	convID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid conversation ID")
		return
	}

	result, err := h.getConversation.Execute(r.Context(), convID, user.ID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Conversation not found")
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// ListConversations godoc
// @Summary      List conversations
// @Description  Returns all conversations belonging to the authenticated user
// @Tags         conversations
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.SwaggerConversationListResponse
// @Failure      401  {object}  response.SwaggerErrorResponse
// @Failure      500  {object}  response.SwaggerErrorResponse
// @Router       /v1/conversations [get]
func (h *ChatHandler) ListConversations(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())

	result, err := h.listConversations.Execute(r.Context(), user.ID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "LIST_ERROR", err.Error())
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// ApproveAction godoc
// @Summary      Approve or reject a pending action
// @Description  Resumes a conversation that is paused waiting for human-in-the-loop approval
// @Tags         conversations
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                                true  "Conversation ID (UUID)"
// @Param        body  body      request.ApprovalRequest               true  "Approval decision"
// @Success      200   {object}  response.SwaggerApprovalResponse
// @Failure      400   {object}  response.SwaggerErrorResponse
// @Failure      401   {object}  response.SwaggerErrorResponse
// @Failure      500   {object}  response.SwaggerErrorResponse
// @Router       /v1/conversations/{id}/approve [post]
func (h *ChatHandler) ApproveAction(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())

	convID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid conversation ID")
		return
	}

	var req request.ApprovalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	result, err := h.approveAction.Execute(r.Context(), chat.ApproveInput{
		ConversationID: convID,
		UserID:         user.ID,
		Approved:       req.Approved,
		Reason:         req.Reason,
	})
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "APPROVAL_ERROR", err.Error())
		return
	}

	response.JSON(w, http.StatusOK, result)
}
