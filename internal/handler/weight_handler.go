package handler

import (
	"encoding/json"
	"net/http"

	"backend/internal/domain"
	"backend/internal/repository"
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
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var req AddWeightRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	rec, err := h.weightRepo.Create(r.Context(), userID, req.Date, req.PreWeight, req.PostWeight)
	if err != nil {
		http.Error(w, "Failed to create weight record", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(rec)
}

func (h *WeightHandler) List(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.PathValue("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	records, err := h.weightRepo.ListByUser(r.Context(), userID)
	if err != nil {
		http.Error(w, "Failed to list weight records", http.StatusInternalServerError)
		return
	}

	if records == nil {
		records = make([]domain.WeightRecord, 0)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(records)
}

func (h *WeightHandler) Delete(w http.ResponseWriter, r *http.Request) {
	recordIDStr := r.PathValue("id")
	recordID, err := uuid.Parse(recordIDStr)
	if err != nil {
		http.Error(w, "Invalid record ID", http.StatusBadRequest)
		return
	}

	if err := h.weightRepo.Delete(r.Context(), recordID); err != nil {
		http.Error(w, "Failed to delete weight record", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *WeightHandler) Update(w http.ResponseWriter, r *http.Request) {
	recordIDStr := r.PathValue("id")
	recordID, err := uuid.Parse(recordIDStr)
	if err != nil {
		http.Error(w, "Invalid record ID", http.StatusBadRequest)
		return
	}

	var req AddWeightRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	rec, err := h.weightRepo.Update(r.Context(), recordID, req.Date, req.PreWeight, req.PostWeight)
	if err != nil {
		http.Error(w, "Failed to update weight record", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rec)
}
