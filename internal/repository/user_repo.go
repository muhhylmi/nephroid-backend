package repository

import (
	"context"
	"errors"
	"fmt"

	"backend/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type userRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, email string) (*domain.User, error) {
	query := `INSERT INTO users (email) VALUES ($1) RETURNING id, email, created_at`
	
	var user domain.User
	err := r.db.QueryRow(ctx, query, email).Scan(&user.ID, &user.Email, &user.CreatedAt)
	if err != nil {
		// pgx specific error mapping can be added here, for now simple error wrapping
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &user, nil
}

func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `SELECT id, email, created_at FROM users WHERE id = $1`
	
	var user domain.User
	err := r.db.QueryRow(ctx, query, id).Scan(&user.ID, &user.Email, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}

	return &user, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `SELECT id, email, created_at FROM users WHERE email = $1`
	
	var user domain.User
	err := r.db.QueryRow(ctx, query, email).Scan(&user.ID, &user.Email, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return &user, nil
}
