package handler

import (
	"encoding/json"
	"net/http"

	"backend/internal/domain"
	"backend/internal/repository"
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
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var req AddLabRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	var customValsBytes []byte
	if req.CustomValues != nil {
		customValsBytes = []byte(req.CustomValues)
	}

	rec, err := h.labRepo.Create(r.Context(), userID, req.Date, req.Kreatinin, req.Ureum, req.Kalium, req.Hb, customValsBytes)
	if err != nil {
		http.Error(w, "Failed to create lab record", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(rec)
}

func (h *LabHandler) List(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.PathValue("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	records, err := h.labRepo.ListByUser(r.Context(), userID)
	if err != nil {
		http.Error(w, "Failed to list lab records", http.StatusInternalServerError)
		return
	}

	if records == nil {
		records = make([]domain.LabRecord, 0)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(records)
}

func (h *LabHandler) Delete(w http.ResponseWriter, r *http.Request) {
	recordIDStr := r.PathValue("id")
	recordID, err := uuid.Parse(recordIDStr)
	if err != nil {
		http.Error(w, "Invalid record ID", http.StatusBadRequest)
		return
	}

	if err := h.labRepo.Delete(r.Context(), recordID); err != nil {
		http.Error(w, "Failed to delete lab record", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *LabHandler) Update(w http.ResponseWriter, r *http.Request) {
	recordIDStr := r.PathValue("id")
	recordID, err := uuid.Parse(recordIDStr)
	if err != nil {
		http.Error(w, "Invalid record ID", http.StatusBadRequest)
		return
	}

	var req AddLabRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	var customValsBytes []byte
	if req.CustomValues != nil {
		customValsBytes = []byte(req.CustomValues)
	}

	rec, err := h.labRepo.Update(r.Context(), recordID, req.Date, req.Kreatinin, req.Ureum, req.Kalium, req.Hb, customValsBytes)
	if err != nil {
		http.Error(w, "Failed to update lab record", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rec)
}
