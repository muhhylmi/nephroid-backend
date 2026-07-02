package response

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"backend/internal/domain"
)

type StandardResponse struct {
	Success  bool        `json:"success"`
	Data     interface{} `json:"data"`
	Messages string      `json:"messages"`
}

type Meta struct {
	Page      int `json:"page"`
	PerPage   int `json:"perPage"`
	TotalPage int `json:"totalPage"`
	TotalData int `json:"totalData"`
}

type PaginatedResponse struct {
	Success  bool        `json:"success"`
	Data     interface{} `json:"data"`
	Messages string      `json:"messages"`
	Meta     Meta        `json:"meta"`
}

// JSON handles standard responses
func JSON(w http.ResponseWriter, status int, data interface{}, messages string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	
	resp := StandardResponse{
		Success:  status >= 200 && status < 300,
		Data:     data,
		Messages: messages,
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("failed to encode json response", "error", err)
	}
}

// JSONError handles standard error responses
func JSONError(w http.ResponseWriter, status int, messages string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	resp := StandardResponse{
		Success:  false,
		Data:     nil,
		Messages: messages,
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("failed to encode json error response", "error", err)
	}
}

// JSONPaginated handles paginated responses
func JSONPaginated(w http.ResponseWriter, status int, data interface{}, page, limit, totalData int, messages string) {
	totalPage := totalData / limit
	if totalData%limit > 0 {
		totalPage++
	}
	if totalPage == 0 {
		totalPage = 1
	}

	resp := PaginatedResponse{
		Success:  status >= 200 && status < 300,
		Data:     data,
		Messages: messages,
		Meta: Meta{
			Page:      page,
			PerPage:   limit,
			TotalPage: totalPage,
			TotalData: totalData,
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("failed to encode json paginated response", "error", err)
	}
}

// Error maps domain errors to HTTP errors using the standard format
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

	JSONError(w, status, msg)
}
