package repository

import (
	"context"
	"fmt"

	"backend/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type sessionRepository struct {
	db *pgxpool.Pool
}

func NewSessionRepository(db *pgxpool.Pool) SessionRepository {
	return &sessionRepository{db: db}
}

func (r *sessionRepository) Create(ctx context.Context, userID uuid.UUID, name string) (*domain.ChatSession, error) {
	query := `INSERT INTO chat_sessions (user_id, name) VALUES ($1, $2) RETURNING id, user_id, name, created_at`
	
	var session domain.ChatSession
	err := r.db.QueryRow(ctx, query, userID, name).Scan(&session.ID, &session.UserID, &session.Name, &session.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat session: %w", err)
	}

	return &session, nil
}

func (r *sessionRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.ChatSession, error) {
	query := `SELECT id, user_id, name, created_at FROM chat_sessions WHERE user_id = $1 ORDER BY created_at DESC`
	
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query chat sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*domain.ChatSession
	for rows.Next() {
		var session domain.ChatSession
		if err := rows.Scan(&session.ID, &session.UserID, &session.Name, &session.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan chat session: %w", err)
		}
		sessions = append(sessions, &session)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return sessions, nil
}

func (r *sessionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM chat_sessions WHERE id = $1`
	
	cmd, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete chat session: %w", err)
	}
	
	if cmd.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}
