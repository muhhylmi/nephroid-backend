package service

import (
	"context"
	"fmt"
	"strings"

	"backend/internal/domain"
	"backend/internal/repository"
	"github.com/google/uuid"
)

type UserService interface {
	RegisterUser(ctx context.Context, email string) (*domain.User, error)
	GetUser(ctx context.Context, id uuid.UUID) (*domain.User, error)
}

type userService struct {
	repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) UserService {
	return &userService{repo: repo}
}

func (s *userService) RegisterUser(ctx context.Context, email string) (*domain.User, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return nil, domain.ErrInvalidInput
	}

	// Check if exists
	existing, err := s.repo.GetByEmail(ctx, email)
	if err != nil && err != domain.ErrNotFound {
		return nil, fmt.Errorf("user service error checking email: %w", err)
	}
	if existing != nil {
		return nil, domain.ErrConflict
	}

	user, err := s.repo.Create(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("user service error creating user: %w", err)
	}

	return user, nil
}

func (s *userService) GetUser(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("user service error getting user: %w", err)
	}
	return user, nil
}
