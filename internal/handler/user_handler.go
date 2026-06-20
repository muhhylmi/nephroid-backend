package handler

import (
	"encoding/json"
	"net/http"

	"backend/internal/service"
	"backend/pkg/response"
	"github.com/google/uuid"
)

type UserHandler struct {
	userService service.UserService
}

func NewUserHandler(userService service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, err)
		return
	}

	user, err := h.userService.RegisterUser(r.Context(), req.Email)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, user)
}

func (h *UserHandler) Get(w http.ResponseWriter, r *http.Request) {
	idParam := r.PathValue("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		response.Error(w, err)
		return
	}

	user, err := h.userService.GetUser(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, user)
}
