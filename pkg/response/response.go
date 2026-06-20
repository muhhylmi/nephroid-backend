package response

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"backend/internal/domain"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			slog.Error("failed to encode json response", "error", err)
		}
	}
}

func Error(w http.ResponseWriter, err error) {
	// Single handling rule: we log the technical error details here at the boundary.
	slog.Error("request failed", "error", err)

	status := http.StatusInternalServerError
	msg := "internal server error"

	if errors.Is(err, domain.ErrNotFound) {
		status = http.StatusNotFound
		msg = "resource not found"
	} else if errors.Is(err, domain.ErrConflict) {
		status = http.StatusConflict
		msg = "resource already exists"
	} else if errors.Is(err, domain.ErrInvalidInput) {
		status = http.StatusBadRequest
		msg = err.Error() // We expose validation message to user
	} else if errors.Is(err, domain.ErrUnauthorized) {
		status = http.StatusUnauthorized
		msg = "unauthorized access"
	}

	JSON(w, status, ErrorResponse{Error: msg})
}
