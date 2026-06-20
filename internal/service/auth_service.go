package service

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"backend/internal/domain"
	"backend/internal/repository"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthService interface {
	Register(ctx context.Context, req domain.RegisterRequest) (*domain.AuthResponse, error)
	Login(ctx context.Context, req domain.LoginRequest) (*domain.AuthResponse, error)
}

type authService struct {
	repo repository.UserRepository
}

func NewAuthService(repo repository.UserRepository) AuthService {
	return &authService{repo: repo}
}

func (s *authService) Register(ctx context.Context, req domain.RegisterRequest) (*domain.AuthResponse, error) {
	email := strings.TrimSpace(strings.ToLower(req.Email))
	if email == "" || req.Password == "" {
		return nil, domain.ErrInvalidInput
	}

	existing, err := s.repo.GetByEmail(ctx, email)
	if err != nil && err != domain.ErrNotFound {
		return nil, fmt.Errorf("auth service error checking email: %w", err)
	}
	if existing != nil {
		return nil, domain.ErrConflict
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("auth service error hashing password: %w", err)
	}

	role := "patient"
	if req.Role != "" {
		role = req.Role
	}

	user, err := s.repo.Create(ctx, email, string(hash), role, req.DialysisFrequency)
	if err != nil {
		return nil, fmt.Errorf("auth service error creating user: %w", err)
	}

	token, err := generateJWT(user.ID.String())
	if err != nil {
		return nil, fmt.Errorf("auth service error generating token: %w", err)
	}

	return &domain.AuthResponse{
		Token: token,
		User:  *user,
	}, nil
}

func (s *authService) Login(ctx context.Context, req domain.LoginRequest) (*domain.AuthResponse, error) {
	email := strings.TrimSpace(strings.ToLower(req.Email))
	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, domain.ErrInvalidInput // Generic auth error
		}
		return nil, fmt.Errorf("auth service error getting user: %w", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		return nil, domain.ErrInvalidInput // Generic auth error
	}

	token, err := generateJWT(user.ID.String())
	if err != nil {
		return nil, fmt.Errorf("auth service error generating token: %w", err)
	}

	return &domain.AuthResponse{
		Token: token,
		User:  *user,
	}, nil
}

func generateJWT(userID string) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "default_secret_for_dev_only"
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(time.Hour * 72).Unix(),
	})
	return token.SignedString([]byte(secret))
}
