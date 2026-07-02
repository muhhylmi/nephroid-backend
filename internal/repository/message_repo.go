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

func (r *messageRepository) Create(ctx context.Context, sessionID uuid.UUID, role string, content string, citations interface{}) (*domain.Message, error) {
	query := `INSERT INTO messages (session_id, role, content, citations) VALUES ($1, $2, $3, $4) RETURNING id, session_id, role, content, citations, created_at`
	
	var msg domain.Message
	err := r.db.QueryRow(ctx, query, sessionID, role, content, citations).Scan(&msg.ID, &msg.SessionID, &msg.Role, &msg.Content, &msg.Citations, &msg.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	return &msg, nil
}

func (r *messageRepository) GetBySessionID(ctx context.Context, sessionID uuid.UUID, page, limit int) ([]*domain.Message, int, error) {
	var total int
	countQuery := `SELECT COUNT(*) FROM messages WHERE session_id = $1`
	err := r.db.QueryRow(ctx, countQuery, sessionID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count messages: %w", err)
	}

	offset := (page - 1) * limit
	// Fetch latest messages first
	query := `SELECT id, session_id, role, content, citations, created_at FROM messages WHERE session_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	
	rows, err := r.db.Query(ctx, query, sessionID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()

	var messages []*domain.Message
	for rows.Next() {
		var msg domain.Message
		if err := rows.Scan(&msg.ID, &msg.SessionID, &msg.Role, &msg.Content, &msg.Citations, &msg.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, &msg)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows iteration error: %w", err)
	}

	// Reverse the slice so it's in chronological order (oldest to newest in this batch)
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, total, nil
}
