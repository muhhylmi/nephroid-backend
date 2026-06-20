package domain

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user in the system
type User struct {
	ID                uuid.UUID `json:"id"`
	Email             string    `json:"email"`
	PasswordHash      string    `json:"-"` // never leak password hash to json
	Role              string    `json:"role"`
	DialysisFrequency string    `json:"dialysis_frequency,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}

// ChatSession represents a chat session between a user and the AI
type ChatSession struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// Message represents a single message in a chat session
type Message struct {
	ID        uuid.UUID `json:"id"`
	SessionID uuid.UUID `json:"session_id"`
	Role      string    `json:"role"` // "user" or "assistant"
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// Auth Types

type RegisterRequest struct {
	Email             string `json:"email"`
	Password          string `json:"password"`
	Role              string `json:"role"`
	DialysisFrequency string `json:"dialysis_frequency,omitempty"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}
