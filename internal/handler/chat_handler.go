package handler

import (
	"encoding/json"
	"net/http"

	"backend/internal/service"
	"backend/pkg/response"
	"github.com/google/uuid"
)

type ChatHandler struct {
	chatService service.ChatService
}

func NewChatHandler(chatService service.ChatService) *ChatHandler {
	return &ChatHandler{chatService: chatService}
}

func (h *ChatHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID string `json:"user_id"`
		Name   string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.JSONError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		response.JSONError(w, http.StatusBadRequest, "Invalid user ID format")
		return
	}

	session, err := h.chatService.CreateSession(r.Context(), userID, req.Name)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, session, "success create session")
}

func (h *ChatHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	userIDParam := r.PathValue("userId")
	userID, err := uuid.Parse(userIDParam)
	if err != nil {
		response.JSONError(w, http.StatusBadRequest, "Invalid user ID format")
		return
	}

	sessions, err := h.chatService.ListUserSessions(r.Context(), userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, sessions, "success retrieve sessions")
}

func (h *ChatHandler) DeleteSession(w http.ResponseWriter, r *http.Request) {
	idParam := r.PathValue("id")
	sessionID, err := uuid.Parse(idParam)
	if err != nil {
		response.JSONError(w, http.StatusBadRequest, "Invalid session ID format")
		return
	}

	err = h.chatService.DeleteSession(r.Context(), sessionID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, nil, "success delete session")
}

func (h *ChatHandler) AddMessage(w http.ResponseWriter, r *http.Request) {
	sessionIDParam := r.PathValue("id")
	sessionID, err := uuid.Parse(sessionIDParam)
	if err != nil {
		response.JSONError(w, http.StatusBadRequest, "Invalid session ID format")
		return
	}

	var req struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.JSONError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	msg, err := h.chatService.AddMessage(r.Context(), sessionID, req.Role, req.Content)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, msg, "success add message")
}

func (h *ChatHandler) ListMessages(w http.ResponseWriter, r *http.Request) {
	sessionIDParam := r.PathValue("id")
	sessionID, err := uuid.Parse(sessionIDParam)
	if err != nil {
		response.JSONError(w, http.StatusBadRequest, "Invalid session ID format")
		return
	}

	messages, err := h.chatService.GetSessionMessages(r.Context(), sessionID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, messages, "success retrieve messages")
}
