package repository

import (
	"context"
	"fmt"

	"backend/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WeightRepository interface {
	Create(ctx context.Context, userID uuid.UUID, date string, preWeight, postWeight float64) (*domain.WeightRecord, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]domain.WeightRecord, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type weightRepository struct {
	db *pgxpool.Pool
}

func NewWeightRepository(db *pgxpool.Pool) WeightRepository {
	return &weightRepository{db: db}
}

func (r *weightRepository) Create(ctx context.Context, userID uuid.UUID, date string, preWeight, postWeight float64) (*domain.WeightRecord, error) {
	query := `INSERT INTO weight_records (user_id, date, pre_weight, post_weight) VALUES ($1, $2, $3, $4) RETURNING id, user_id, date, pre_weight, post_weight, created_at`
	
	var rec domain.WeightRecord
	err := r.db.QueryRow(ctx, query, userID, date, preWeight, postWeight).Scan(&rec.ID, &rec.UserID, &rec.Date, &rec.PreWeight, &rec.PostWeight, &rec.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create weight record: %w", err)
	}

	return &rec, nil
}

func (r *weightRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]domain.WeightRecord, error) {
	query := `SELECT id, user_id, date, pre_weight, post_weight, created_at FROM weight_records WHERE user_id = $1 ORDER BY created_at ASC`
	
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query weight records: %w", err)
	}
	defer rows.Close()

	var records []domain.WeightRecord
	for rows.Next() {
		var rec domain.WeightRecord
		if err := rows.Scan(&rec.ID, &rec.UserID, &rec.Date, &rec.PreWeight, &rec.PostWeight, &rec.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan weight record: %w", err)
		}
		records = append(records, rec)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return records, nil
}

func (r *weightRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM weight_records WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete weight record: %w", err)
	}
	return nil
}
