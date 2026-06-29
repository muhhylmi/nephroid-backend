package repository

import (
	"context"
	"encoding/json"
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

func (r *userRepository) Create(ctx context.Context, email, passwordHash, role, dialysisFrequency string) (*domain.User, error) {
	query := `INSERT INTO users (email, password_hash, role, dialysis_frequency) VALUES ($1, $2, $3, $4) RETURNING id, email, password_hash, role, dialysis_frequency, target_dry_weight, lab_parameters, created_at`
	
	var user domain.User
	var freq *string
	var labParams *json.RawMessage
	err := r.db.QueryRow(ctx, query, email, passwordHash, role, dialysisFrequency).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Role, &freq, &user.TargetDryWeight, &labParams, &user.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	if freq != nil {
		user.DialysisFrequency = *freq
	}
	if labParams != nil {
		user.LabParameters = *labParams
	}

	return &user, nil
}

func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `SELECT id, email, password_hash, role, dialysis_frequency, target_dry_weight, lab_parameters, created_at FROM users WHERE id = $1`
	
	var user domain.User
	var freq *string
	var labParams *json.RawMessage
	err := r.db.QueryRow(ctx, query, id).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Role, &freq, &user.TargetDryWeight, &labParams, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}
	if freq != nil {
		user.DialysisFrequency = *freq
	}
	if labParams != nil {
		user.LabParameters = *labParams
	}

	return &user, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `SELECT id, email, password_hash, role, dialysis_frequency, target_dry_weight, lab_parameters, created_at FROM users WHERE email = $1`
	
	var user domain.User
	var freq *string
	var labParams *json.RawMessage
	err := r.db.QueryRow(ctx, query, email).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Role, &freq, &user.TargetDryWeight, &labParams, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	if freq != nil {
		user.DialysisFrequency = *freq
	}
	if labParams != nil {
		user.LabParameters = *labParams
	}

	return &user, nil
}

func (r *userRepository) UpdateProfile(ctx context.Context, id uuid.UUID, req domain.UpdateProfileRequest) (*domain.User, error) {
	query := `UPDATE users SET email = $1, role = $2, dialysis_frequency = $3, target_dry_weight = $4, lab_parameters = $5 WHERE id = $6 RETURNING id, email, password_hash, role, dialysis_frequency, target_dry_weight, lab_parameters, created_at`
	
	var user domain.User
	var freq *string
	var labParams *json.RawMessage
	err := r.db.QueryRow(ctx, query, req.Email, req.Role, req.DialysisFrequency, req.TargetDryWeight, req.LabParameters, id).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Role, &freq, &user.TargetDryWeight, &labParams, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to update user: %w", err)
	}
	if freq != nil {
		user.DialysisFrequency = *freq
	}
	if labParams != nil {
		user.LabParameters = *labParams
	}

	return &user, nil
}
