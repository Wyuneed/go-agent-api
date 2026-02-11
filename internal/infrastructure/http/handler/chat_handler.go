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

// POST /v1/chat/completions - OpenAI-compatible endpoint
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

// POST /v1/conversations
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

// POST /v1/conversations/:id/messages
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

// GET /v1/conversations/:id
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

// GET /v1/conversations
func (h *ChatHandler) ListConversations(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())

	result, err := h.listConversations.Execute(r.Context(), user.ID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "LIST_ERROR", err.Error())
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// POST /v1/conversations/:id/approve
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
