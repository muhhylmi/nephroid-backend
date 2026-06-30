package repository

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"backend/internal/domain"
	"github.com/google/uuid"
	"github.com/qdrant/go-client/qdrant"
)

// KnowledgeRepository handles persistence of knowledge chunks in Qdrant.
type KnowledgeRepository interface {
	InsertChunk(ctx context.Context, content, timestamp string, embedding []float32) (*domain.KnowledgeChunk, error)
	CountChunks(ctx context.Context) (int, error)
	ClearAll(ctx context.Context) error
	SearchChunks(ctx context.Context, embedding []float32, limit int) ([]domain.KnowledgeChunk, error)
	EnsureCollectionExists(ctx context.Context) error
}

type knowledgeRepository struct {
	client         *qdrant.Client
	collectionName string
}

func NewKnowledgeRepository(client *qdrant.Client) KnowledgeRepository {
	return &knowledgeRepository{
		client:         client,
		collectionName: "knowledge_base",
	}
}

// EnsureCollectionExists creates the Qdrant collection if it doesn't exist.
func (r *knowledgeRepository) EnsureCollectionExists(ctx context.Context) error {
	exists, err := r.client.CollectionExists(ctx, r.collectionName)
	if err != nil {
		return fmt.Errorf("failed to check collection existence: %w", err)
	}

	if !exists {
		vectorSize := 384 // Default for sentence-transformers/all-MiniLM-L6-v2
		if sizeStr := os.Getenv("QDRANT_VECTOR_SIZE"); sizeStr != "" {
			if s, err := strconv.Atoi(sizeStr); err == nil {
				vectorSize = s
			}
		}

		err = r.client.CreateCollection(ctx, &qdrant.CreateCollection{
			CollectionName: r.collectionName,
			VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
				Size:     uint64(vectorSize),
				Distance: qdrant.Distance_Cosine,
			}),
		})
		if err != nil {
			return fmt.Errorf("failed to create collection: %w", err)
		}
	}
	return nil
}

// InsertChunk inserts a knowledge chunk and its embedding into Qdrant.
func (r *knowledgeRepository) InsertChunk(ctx context.Context, content, timestamp string, embedding []float32) (*domain.KnowledgeChunk, error) {
	id := uuid.New()
	createdAt := time.Now()

	points := []*qdrant.PointStruct{
		{
			Id:      qdrant.NewIDUUID(id.String()),
			Vectors: qdrant.NewVectors(embedding...),
			Payload: qdrant.NewValueMap(map[string]any{
				"content":    content,
				"timestamp":  timestamp,
				"created_at": createdAt.Format(time.RFC3339),
			}),
		},
	}

	_, err := r.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: r.collectionName,
		Points:         points,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to insert chunk to qdrant: %w", err)
	}

	return &domain.KnowledgeChunk{
		ID:        id,
		Content:   content,
		Timestamp: timestamp,
		CreatedAt: createdAt,
	}, nil
}

// CountChunks returns the total number of knowledge chunks in Qdrant.
func (r *knowledgeRepository) CountChunks(ctx context.Context) (int, error) {
	count, err := r.client.Count(ctx, &qdrant.CountPoints{
		CollectionName: r.collectionName,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to count chunks: %w", err)
	}
	return int(count), nil
}

// ClearAll removes all knowledge chunks — used before re-ingestion.
func (r *knowledgeRepository) ClearAll(ctx context.Context) error {
	err := r.client.DeleteCollection(ctx, r.collectionName)
	if err != nil {
		return fmt.Errorf("failed to delete collection: %w", err)
	}

	// Recreate it empty
	return r.EnsureCollectionExists(ctx)
}

// SearchChunks performs a vector similarity search in Qdrant.
func (r *knowledgeRepository) SearchChunks(ctx context.Context, embedding []float32, limit int) ([]domain.KnowledgeChunk, error) {
	searchResult, err := r.client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: r.collectionName,
		Query:          qdrant.NewQuery(embedding...),
		Limit:          func() *uint64 { l := uint64(limit); return &l }(),
		WithPayload:    qdrant.NewWithPayload(true),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to search knowledge chunks: %w", err)
	}

	var chunks []domain.KnowledgeChunk
	for _, scoredPoint := range searchResult {
		payload := scoredPoint.Payload
		
		content := ""
		if val, ok := payload["content"]; ok {
			content = val.GetStringValue()
		}
		
		timestamp := ""
		if val, ok := payload["timestamp"]; ok {
			timestamp = val.GetStringValue()
		}

		createdAt := time.Now()
		if val, ok := payload["created_at"]; ok {
			parsedTime, err := time.Parse(time.RFC3339, val.GetStringValue())
			if err == nil {
				createdAt = parsedTime
			}
		}

		pointID := scoredPoint.Id.GetUuid()
		parsedID, _ := uuid.Parse(pointID)

		chunks = append(chunks, domain.KnowledgeChunk{
			ID:        parsedID,
			Content:   content,
			Timestamp: timestamp,
			CreatedAt: createdAt,
		})
	}

	return chunks, nil
}
