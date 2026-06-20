package repository

import (
	"context"
	"fmt"

	"backend/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type messageRepository struct {
	db *pgxpool.Pool
}

func NewMessageRepository(db *pgxpool.Pool) MessageRepository {
	return &messageRepository{db: db}
}

func (r *messageRepository) Create(ctx context.Context, sessionID uuid.UUID, role string, content string) (*domain.Message, error) {
	query := `INSERT INTO messages (session_id, role, content) VALUES ($1, $2, $3) RETURNING id, session_id, role, content, created_at`
	
	var msg domain.Message
	err := r.db.QueryRow(ctx, query, sessionID, role, content).Scan(&msg.ID, &msg.SessionID, &msg.Role, &msg.Content, &msg.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	return &msg, nil
}

func (r *messageRepository) GetBySessionID(ctx context.Context, sessionID uuid.UUID) ([]*domain.Message, error) {
	query := `SELECT id, session_id, role, content, created_at FROM messages WHERE session_id = $1 ORDER BY created_at ASC`
	
	rows, err := r.db.Query(ctx, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()

	var messages []*domain.Message
	for rows.Next() {
		var msg domain.Message
		if err := rows.Scan(&msg.ID, &msg.SessionID, &msg.Role, &msg.Content, &msg.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, &msg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return messages, nil
}
