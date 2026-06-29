package repository

import (
	"context"
	"fmt"

	"backend/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LabRepository interface {
	Create(ctx context.Context, userID uuid.UUID, date string, kreatinin, ureum, kalium, hb float64) (*domain.LabRecord, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]domain.LabRecord, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type labRepository struct {
	db *pgxpool.Pool
}

func NewLabRepository(db *pgxpool.Pool) LabRepository {
	return &labRepository{db: db}
}

func (r *labRepository) Create(ctx context.Context, userID uuid.UUID, date string, kreatinin, ureum, kalium, hb float64) (*domain.LabRecord, error) {
	query := `INSERT INTO lab_records (user_id, date, kreatinin, ureum, kalium, hb) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, user_id, date, kreatinin, ureum, kalium, hb, created_at`
	
	var rec domain.LabRecord
	err := r.db.QueryRow(ctx, query, userID, date, kreatinin, ureum, kalium, hb).Scan(&rec.ID, &rec.UserID, &rec.Date, &rec.Kreatinin, &rec.Ureum, &rec.Kalium, &rec.Hb, &rec.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create lab record: %w", err)
	}

	return &rec, nil
}

func (r *labRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]domain.LabRecord, error) {
	query := `SELECT id, user_id, date, kreatinin, ureum, kalium, hb, created_at FROM lab_records WHERE user_id = $1 ORDER BY created_at ASC`
	
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query lab records: %w", err)
	}
	defer rows.Close()

	var records []domain.LabRecord
	for rows.Next() {
		var rec domain.LabRecord
		if err := rows.Scan(&rec.ID, &rec.UserID, &rec.Date, &rec.Kreatinin, &rec.Ureum, &rec.Kalium, &rec.Hb, &rec.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan lab record: %w", err)
		}
		records = append(records, rec)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return records, nil
}

func (r *labRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM lab_records WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete lab record: %w", err)
	}
	return nil
}
