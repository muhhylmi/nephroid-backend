package repository

import (
	"context"

	"backend/internal/domain"
	"github.com/google/uuid"
)

type UserRepository interface {
	Create(ctx context.Context, email, passwordHash, role, dialysisFrequency string) (*domain.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	UpdateProfile(ctx context.Context, id uuid.UUID, req domain.UpdateProfileRequest) (*domain.User, error)
}

type SessionRepository interface {
	Create(ctx context.Context, userID uuid.UUID, name string) (*domain.ChatSession, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.ChatSession, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type MessageRepository interface {
	Create(ctx context.Context, sessionID uuid.UUID, role string, content string) (*domain.Message, error)
	GetBySessionID(ctx context.Context, sessionID uuid.UUID, page, limit int) ([]*domain.Message, int, error)
}
