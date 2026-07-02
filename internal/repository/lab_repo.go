package repository

import (
	"context"
	"fmt"

	"backend/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LabRepository interface {
	Create(ctx context.Context, userID uuid.UUID, date string, kreatinin, ureum, kalium, hb float64, customValues []byte) (*domain.LabRecord, error)
	ListByUser(ctx context.Context, userID uuid.UUID, page, limit int, sortBy, sortDir string) ([]domain.LabRecord, int, error)
	Update(ctx context.Context, id uuid.UUID, date string, kreatinin, ureum, kalium, hb float64, customValues []byte) (*domain.LabRecord, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type labRepository struct {
	db *pgxpool.Pool
}

func NewLabRepository(db *pgxpool.Pool) LabRepository {
	return &labRepository{db: db}
}

func (r *labRepository) Create(ctx context.Context, userID uuid.UUID, date string, kreatinin, ureum, kalium, hb float64, customValues []byte) (*domain.LabRecord, error) {
	query := `INSERT INTO lab_records (user_id, date, kreatinin, ureum, kalium, hb, custom_values) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, user_id, date, kreatinin, ureum, kalium, hb, custom_values, created_at`
	
	var rec domain.LabRecord
	err := r.db.QueryRow(ctx, query, userID, date, kreatinin, ureum, kalium, hb, customValues).Scan(&rec.ID, &rec.UserID, &rec.Date, &rec.Kreatinin, &rec.Ureum, &rec.Kalium, &rec.Hb, &rec.CustomValues, &rec.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create lab record: %w", err)
	}

	return &rec, nil
}

func (r *labRepository) ListByUser(ctx context.Context, userID uuid.UUID, page, limit int, sortBy, sortDir string) ([]domain.LabRecord, int, error) {
	var total int
	countQuery := `SELECT COUNT(*) FROM lab_records WHERE user_id = $1`
	err := r.db.QueryRow(ctx, countQuery, userID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count lab records: %w", err)
	}

	// Validate sortDir
	if sortDir != "asc" && sortDir != "desc" {
		sortDir = "desc" // default
	}

	// Validate sortBy to prevent SQL injection
	allowedSortColumns := map[string]string{
		"date":      "date",
		"kreatinin": "kreatinin",
		"ureum":     "ureum",
		"kalium":    "kalium",
		"hb":        "hb",
	}

	orderColumn, exists := allowedSortColumns[sortBy]
	if !exists {
		orderColumn = "date" // default
	}

	offset := (page - 1) * limit
	query := fmt.Sprintf(`SELECT id, user_id, date, kreatinin, ureum, kalium, hb, custom_values, created_at FROM lab_records WHERE user_id = $1 ORDER BY %s %s, created_at DESC LIMIT $2 OFFSET $3`, orderColumn, sortDir)
	
	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query lab records: %w", err)
	}
	defer rows.Close()

	var records []domain.LabRecord
	for rows.Next() {
		var rec domain.LabRecord
		if err := rows.Scan(&rec.ID, &rec.UserID, &rec.Date, &rec.Kreatinin, &rec.Ureum, &rec.Kalium, &rec.Hb, &rec.CustomValues, &rec.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan lab record: %w", err)
		}
		records = append(records, rec)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows iteration error: %w", err)
	}

	return records, total, nil
}

func (r *labRepository) Update(ctx context.Context, id uuid.UUID, date string, kreatinin, ureum, kalium, hb float64, customValues []byte) (*domain.LabRecord, error) {
	query := `UPDATE lab_records SET date = $1, kreatinin = $2, ureum = $3, kalium = $4, hb = $5, custom_values = $6 WHERE id = $7 RETURNING id, user_id, date, kreatinin, ureum, kalium, hb, custom_values, created_at`
	
	var rec domain.LabRecord
	err := r.db.QueryRow(ctx, query, date, kreatinin, ureum, kalium, hb, customValues, id).Scan(&rec.ID, &rec.UserID, &rec.Date, &rec.Kreatinin, &rec.Ureum, &rec.Kalium, &rec.Hb, &rec.CustomValues, &rec.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to update lab record: %w", err)
	}

	return &rec, nil
}

func (r *labRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM lab_records WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete lab record: %w", err)
	}
	return nil
}
