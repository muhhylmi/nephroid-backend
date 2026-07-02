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

type WeightHandler struct {
	weightRepo repository.WeightRepository
}

func NewWeightHandler(weightRepo repository.WeightRepository) *WeightHandler {
	return &WeightHandler{weightRepo: weightRepo}
}

type AddWeightRequest struct {
	Date       string  `json:"date"`
	PreWeight  float64 `json:"preWeight"`
	PostWeight float64 `json:"postWeight"`
}

func (h *WeightHandler) Create(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.PathValue("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.JSONError(w, http.StatusBadRequest, "Invalid user ID format")
		return
	}

	var req AddWeightRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.JSONError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	rec, err := h.weightRepo.Create(r.Context(), userID, req.Date, req.PreWeight, req.PostWeight)
	if err != nil {
		response.JSONError(w, http.StatusInternalServerError, "Failed to create weight record")
		return
	}

	response.JSON(w, http.StatusCreated, rec, "success create weight record")
}

func (h *WeightHandler) List(w http.ResponseWriter, r *http.Request) {
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

	records, totalData, err := h.weightRepo.ListByUser(r.Context(), userID, page, limit, sortBy, sortDir)
	if err != nil {
		response.JSONError(w, http.StatusInternalServerError, "Failed to list weight records")
		return
	}

	if records == nil {
		records = make([]domain.WeightRecord, 0)
	}

	response.JSONPaginated(w, http.StatusOK, records, page, limit, totalData, "success retrieve weight records")
}

func (h *WeightHandler) Delete(w http.ResponseWriter, r *http.Request) {
	recordIDStr := r.PathValue("id")
	recordID, err := uuid.Parse(recordIDStr)
	if err != nil {
		response.JSONError(w, http.StatusBadRequest, "Invalid record ID format")
		return
	}

	if err := h.weightRepo.Delete(r.Context(), recordID); err != nil {
		response.JSONError(w, http.StatusInternalServerError, "Failed to delete weight record")
		return
	}

	response.JSON(w, http.StatusOK, nil, "success delete weight record")
}

func (h *WeightHandler) Update(w http.ResponseWriter, r *http.Request) {
	recordIDStr := r.PathValue("id")
	recordID, err := uuid.Parse(recordIDStr)
	if err != nil {
		response.JSONError(w, http.StatusBadRequest, "Invalid record ID format")
		return
	}

	var req AddWeightRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.JSONError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	rec, err := h.weightRepo.Update(r.Context(), recordID, req.Date, req.PreWeight, req.PostWeight)
	if err != nil {
		response.JSONError(w, http.StatusInternalServerError, "Failed to update weight record")
		return
	}

	response.JSON(w, http.StatusOK, rec, "success update weight record")
}
