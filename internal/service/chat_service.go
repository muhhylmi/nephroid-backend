package service

import (
	"context"
	"fmt"
	"strings"

	"backend/internal/domain"
	"backend/internal/repository"
	"github.com/google/uuid"
)

type ChatService interface {
	CreateSession(ctx context.Context, userID uuid.UUID, name string) (*domain.ChatSession, error)
	ListUserSessions(ctx context.Context, userID uuid.UUID) ([]*domain.ChatSession, error)
	DeleteSession(ctx context.Context, sessionID uuid.UUID) error
	AddMessage(ctx context.Context, sessionID uuid.UUID, role string, content string, citations interface{}) (*domain.Message, error)
	GetSessionMessages(ctx context.Context, sessionID uuid.UUID, page, limit int) ([]*domain.Message, int, error)
}

type chatService struct {
	sessionRepo repository.SessionRepository
	messageRepo repository.MessageRepository
}

func NewChatService(sessionRepo repository.SessionRepository, messageRepo repository.MessageRepository) ChatService {
	return &chatService{
		sessionRepo: sessionRepo,
		messageRepo: messageRepo,
	}
}

func (s *chatService) CreateSession(ctx context.Context, userID uuid.UUID, name string) (*domain.ChatSession, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("%w: session name cannot be empty", domain.ErrInvalidInput)
	}

	session, err := s.sessionRepo.Create(ctx, userID, name)
	if err != nil {
		return nil, fmt.Errorf("chat service error creating session: %w", err)
	}

	return session, nil
}

func (s *chatService) ListUserSessions(ctx context.Context, userID uuid.UUID) ([]*domain.ChatSession, error) {
	sessions, err := s.sessionRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("chat service error listing sessions: %w", err)
	}

	// ensure we never return nil slice for JSON marshalling (idiomatic Go return empty slice)
	if sessions == nil {
		sessions = []*domain.ChatSession{}
	}

	return sessions, nil
}

func (s *chatService) DeleteSession(ctx context.Context, sessionID uuid.UUID) error {
	err := s.sessionRepo.Delete(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("chat service error deleting session: %w", err)
	}
	return nil
}

func (s *chatService) AddMessage(ctx context.Context, sessionID uuid.UUID, role string, content string, citations interface{}) (*domain.Message, error) {
	role = strings.TrimSpace(strings.ToLower(role))
	content = strings.TrimSpace(content)

	if role != "user" && role != "assistant" {
		return nil, fmt.Errorf("%w: role must be user or assistant", domain.ErrInvalidInput)
	}
	if content == "" {
		return nil, fmt.Errorf("%w: message content cannot be empty", domain.ErrInvalidInput)
	}

	msg, err := s.messageRepo.Create(ctx, sessionID, role, content, citations)
	if err != nil {
		return nil, fmt.Errorf("chat service error creating message: %w", err)
	}

	return msg, nil
}

func (s *chatService) GetSessionMessages(ctx context.Context, sessionID uuid.UUID, page, limit int) ([]*domain.Message, int, error) {
	messages, total, err := s.messageRepo.GetBySessionID(ctx, sessionID, page, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("chat service error listing messages: %w", err)
	}

	if messages == nil {
		messages = []*domain.Message{}
	}

	return messages, total, nil
}
