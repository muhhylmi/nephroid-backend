package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"backend/internal/domain"
	"backend/internal/repository"
	"backend/pkg/response"
	"github.com/google/uuid"
)

type LabHandler struct {
	labRepo repository.LabRepository
}

func NewLabHandler(labRepo repository.LabRepository) *LabHandler {
	return &LabHandler{labRepo: labRepo}
}

type AddLabRequest struct {
	Date         string          `json:"date"`
	Kreatinin    float64         `json:"kreatinin"`
	Ureum        float64         `json:"ureum"`
	Kalium       float64         `json:"kalium"`
	Hb           float64         `json:"hb"`
	CustomValues json.RawMessage `json:"custom_values,omitempty"`
}

func (h *LabHandler) Create(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.PathValue("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.JSONError(w, http.StatusBadRequest, "Invalid user ID format")
		return
	}

	var req AddLabRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.JSONError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	var customValsBytes []byte
	if req.CustomValues != nil {
		customValsBytes = []byte(req.CustomValues)
	}

	rec, err := h.labRepo.Create(r.Context(), userID, req.Date, req.Kreatinin, req.Ureum, req.Kalium, req.Hb, customValsBytes)
	if err != nil {
		response.JSONError(w, http.StatusInternalServerError, "Failed to create lab record")
		return
	}

	response.JSON(w, http.StatusCreated, rec, "success create lab record")
}

func (h *LabHandler) List(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.PathValue("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.JSONError(w, http.StatusBadRequest, "Invalid user ID format")
		return
	}

	page := 1
	limit := 10
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	
	sortBy := r.URL.Query().Get("sortBy")
	sortDir := r.URL.Query().Get("sortDir")

	records, totalData, err := h.labRepo.ListByUser(r.Context(), userID, page, limit, sortBy, sortDir)
	if err != nil {
		response.JSONError(w, http.StatusInternalServerError, "Failed to list lab records")
		return
	}

	if records == nil {
		records = make([]domain.LabRecord, 0)
	}

	response.JSONPaginated(w, http.StatusOK, records, page, limit, totalData, "success retrieve lab records")
}

func (h *LabHandler) Delete(w http.ResponseWriter, r *http.Request) {
	recordIDStr := r.PathValue("id")
	recordID, err := uuid.Parse(recordIDStr)
	if err != nil {
		response.JSONError(w, http.StatusBadRequest, "Invalid record ID format")
		return
	}

	if err := h.labRepo.Delete(r.Context(), recordID); err != nil {
		response.JSONError(w, http.StatusInternalServerError, "Failed to delete lab record")
		return
	}

	response.JSON(w, http.StatusOK, nil, "success delete lab record")
}

func (h *LabHandler) Update(w http.ResponseWriter, r *http.Request) {
	recordIDStr := r.PathValue("id")
	recordID, err := uuid.Parse(recordIDStr)
	if err != nil {
		response.JSONError(w, http.StatusBadRequest, "Invalid record ID format")
		return
	}

	var req AddLabRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.JSONError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	var customValsBytes []byte
	if req.CustomValues != nil {
		customValsBytes = []byte(req.CustomValues)
	}

	rec, err := h.labRepo.Update(r.Context(), recordID, req.Date, req.Kreatinin, req.Ureum, req.Kalium, req.Hb, customValsBytes)
	if err != nil {
		response.JSONError(w, http.StatusInternalServerError, "Failed to update lab record")
		return
	}

	response.JSON(w, http.StatusOK, rec, "success update lab record")
}
