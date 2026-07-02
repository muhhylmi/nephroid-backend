package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"backend/internal/service"
	"backend/pkg/response"
	"github.com/google/uuid"
)

type RagHandler struct {
	ragService *service.RagService
	aiService  service.AiService
}

func NewRagHandler(ragService *service.RagService, aiService service.AiService) *RagHandler {
	return &RagHandler{
		ragService: ragService,
		aiService:  aiService,
	}
}

// GenerateKnowledge handles POST /api/rag/generate-knowledge
// It reads the raw chat file, filters and cleans it, then writes based-knowledge.txt.
func (h *RagHandler) GenerateKnowledge(w http.ResponseWriter, r *http.Request) {
	count, err := h.ragService.GenerateKnowledge()
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, map[string]interface{}{
		"entries": count,
	}, "Knowledge base generated successfully")
}

// IngestKnowledge handles POST /api/rag/ingest
// It starts background ingestion of based-knowledge.txt into the vector database
// and returns the process info immediately.
func (h *RagHandler) IngestKnowledge(w http.ResponseWriter, r *http.Request) {
	process, err := h.ragService.StartIngestion()
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusAccepted, process, "Ingestion process started")
}

// GetIngestStatus handles GET /api/rag/ingest/{processId}
// Returns the current status of a background ingestion process.
func (h *RagHandler) GetIngestStatus(w http.ResponseWriter, r *http.Request) {
	processID := r.PathValue("processId")
	if processID == "" {
		response.JSONError(w, http.StatusBadRequest, "processId is required")
		return
	}

	process, err := h.ragService.GetIngestStatus(processID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, process, "success retrieve ingest status")
}

// HandleChat handles POST /api/rag/chat
func (h *RagHandler) HandleChat(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SessionID string `json:"session_id"`
		Message   string `json:"message"`
	}

	importJSON := json.NewDecoder(r.Body)
	if err := importJSON.Decode(&req); err != nil {
		response.JSONError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	sessionUUID, err := uuid.Parse(req.SessionID)
	if err != nil {
		response.JSONError(w, http.StatusBadRequest, "Invalid session ID format")
		return
	}

	reply, chunks, err := h.aiService.Chat(r.Context(), sessionUUID, req.Message)
	if err != nil {
		response.Error(w, err)
		return
	}

	resp := map[string]interface{}{
		"reply":   reply,
		"sources": chunks,
	}

	response.JSON(w, http.StatusOK, resp, "success chat response")
}

// HandleChatStream handles POST /api/rag/chat/stream using Server-Sent Events (SSE)
func (h *RagHandler) HandleChatStream(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SessionID string `json:"session_id"`
		Message   string `json:"message"`
	}

	importJSON := json.NewDecoder(r.Body)
	if err := importJSON.Decode(&req); err != nil {
		response.JSONError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	sessionUUID, err := uuid.Parse(req.SessionID)
	if err != nil {
		response.JSONError(w, http.StatusBadRequest, "Invalid session ID format")
		return
	}

	// Setup SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Ensure the writer supports flushing
	flusher, ok := w.(http.Flusher)
	if !ok {
		response.JSONError(w, http.StatusInternalServerError, "Streaming unsupported")
		return
	}

	chunkChan := make(chan string, 10)

	sources, err := h.aiService.ChatStream(r.Context(), sessionUUID, req.Message, chunkChan)
	if err != nil {
		// Can't use response.Error because headers might already be written, but we haven't flushed yet.
		// We'll write an error event.
		errData, _ := json.Marshal(map[string]string{"error": err.Error()})
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", errData)
		flusher.Flush()
		return
	}

	// 1. Send sources immediately
	sourcesData, _ := json.Marshal(sources)
	fmt.Fprintf(w, "event: sources\ndata: %s\n\n", sourcesData)
	flusher.Flush()

	// 2. Stream message chunks
	for textChunk := range chunkChan {
		chunkData, _ := json.Marshal(map[string]string{"chunk": textChunk})
		fmt.Fprintf(w, "event: message\ndata: %s\n\n", chunkData)
		flusher.Flush()
	}

	// 3. Send done event
	fmt.Fprintf(w, "event: done\ndata: {}\n\n")
	flusher.Flush()
}
