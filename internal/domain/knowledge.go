package domain

import (
	"time"

	"github.com/google/uuid"
)

// KnowledgeChunk represents a single knowledge entry stored in the vector database.
type KnowledgeChunk struct {
	ID        uuid.UUID `json:"id"`
	Content   string    `json:"content"`
	Timestamp string    `json:"timestamp,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// IngestProcess tracks the status of a background knowledge ingestion job.
type IngestProcess struct {
	ProcessID string `json:"process_id"`
	Status    string `json:"status"` // "pending", "processing", "finished", "failed"
	TotalRow  int    `json:"total_row"`
	Ingested  int    `json:"ingested"`
	Error     string `json:"error,omitempty"`
}
