package service

import (
	"context"
	"fmt"
	"os"
	"strings"

	"backend/internal/domain"
	"backend/internal/repository"

	"github.com/google/generative-ai-go/genai"
	"github.com/google/uuid"
	"google.golang.org/api/option"
)

type AiService interface {
	Chat(ctx context.Context, sessionID uuid.UUID, message string) (string, []domain.KnowledgeChunk, error)
	ChatStream(ctx context.Context, sessionID uuid.UUID, message string, chunkChan chan<- string) ([]domain.KnowledgeChunk, error)
}

type aiService struct {
	knowledgeRepo repository.KnowledgeRepository
	chatService   ChatService
}

func NewAiService(knowledgeRepo repository.KnowledgeRepository, chatService ChatService) AiService {
	return &aiService{
		knowledgeRepo: knowledgeRepo,
		chatService:   chatService,
	}
}

// Chat handles a user message, retrieves context, and calls Gemini to generate a response.
func (s *aiService) Chat(ctx context.Context, sessionID uuid.UUID, message string) (string, []domain.KnowledgeChunk, error) {
	if strings.TrimSpace(message) == "" {
		return "", nil, fmt.Errorf("%w: message cannot be empty", domain.ErrInvalidInput)
	}

	// 1. Save user message to database
	_, err := s.chatService.AddMessage(ctx, sessionID, "user", message, nil)
	if err != nil {
		return "", nil, fmt.Errorf("failed to save user message: %w", err)
	}

	// 2. Initialize Gemini client
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return "", nil, fmt.Errorf("GEMINI_API_KEY is not set in environment")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return "", nil, fmt.Errorf("failed to create gemini client: %w", err)
	}
	defer client.Close()

	// 3. Generate query embedding using Hugging Face
	embedding, err := GenerateHFEmbedding(ctx, message)
	if err != nil {
		return "", nil, fmt.Errorf("failed to embed query with HF: %w", err)
	}

	// 4. Retrieve context from knowledge base (Qdrant)
	// Limit to top 5 chunks
	chunks, err := s.knowledgeRepo.SearchChunks(ctx, embedding, 5)
	if err != nil {
		return "", nil, fmt.Errorf("failed to search knowledge base: %w", err)
	}

	// 5. Prepare context string
	var contextBuilder strings.Builder
	for i, chunk := range chunks {
		contextBuilder.WriteString(fmt.Sprintf("\n--- Context %d (Time: %s) ---\n", i+1, chunk.Timestamp))
		contextBuilder.WriteString(chunk.Content)
		contextBuilder.WriteString("\n")
	}
	contextStr := contextBuilder.String()
	// Use gemini-1.5-flash for fast chat responses
	model := client.GenerativeModel("gemini-2.5-flash")

	systemInstruction := `You are an empathetic, knowledgeable assistant specializing in kidney disease (GGK/CKD), Hemodialysis (HD), and CAPD.
Use the provided context to answer the user's question accurately. If the context does not contain the answer, say that you don't know based on the provided knowledge base, but you can offer general advice.
Keep your responses helpful, supportive, and grounded in the provided facts.`

	prompt := fmt.Sprintf("%s\n\nContext information:\n%s\n\nUser Question: %s", systemInstruction, contextStr, message)

	// 6. Generate Response
	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate response from gemini: %w", err)
	}

	var aiReplyBuilder strings.Builder
	for _, cand := range resp.Candidates {
		if cand.Content != nil {
			for _, part := range cand.Content.Parts {
				if txt, ok := part.(genai.Text); ok {
					aiReplyBuilder.WriteString(string(txt))
				}
			}
		}
	}
	aiReply := strings.TrimSpace(aiReplyBuilder.String())

	if aiReply == "" {
		aiReply = "Maaf, saya tidak dapat menghasilkan respon saat ini."
	}

	// 7. Save AI response to database
	_, err = s.chatService.AddMessage(ctx, sessionID, "assistant", aiReply, chunks)
	if err != nil {
		return "", nil, fmt.Errorf("failed to save assistant message: %w", err)
	}

	return aiReply, chunks, nil
}

// ChatStream handles a user message and streams the AI response back via a channel.
func (s *aiService) ChatStream(ctx context.Context, sessionID uuid.UUID, message string, chunkChan chan<- string) ([]domain.KnowledgeChunk, error) {
	if strings.TrimSpace(message) == "" {
		return nil, fmt.Errorf("%w: message cannot be empty", domain.ErrInvalidInput)
	}

	// 1. Save user message to database
	_, err := s.chatService.AddMessage(ctx, sessionID, "user", message, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to save user message: %w", err)
	}

	// 2. Generate query embedding using Hugging Face
	embedding, err := GenerateHFEmbedding(ctx, message)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query with HF: %w", err)
	}

	// 3. Retrieve context from knowledge base (Qdrant)
	chunks, err := s.knowledgeRepo.SearchChunks(ctx, embedding, 5)
	if err != nil {
		return nil, fmt.Errorf("failed to search knowledge base: %w", err)
	}

	// 4. Prepare context string
	var contextBuilder strings.Builder
	for i, chunk := range chunks {
		contextBuilder.WriteString(fmt.Sprintf("\n--- Context %d (Time: %s) ---\n", i+1, chunk.Timestamp))
		contextBuilder.WriteString(chunk.Content)
		contextBuilder.WriteString("\n")
	}
	contextStr := contextBuilder.String()

	// 5. Initialize Gemini client
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY is not set in environment")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create gemini client: %w", err)
	}

	model := client.GenerativeModel("gemini-2.5-flash")
	systemInstruction := `You are an empathetic, knowledgeable assistant specializing in kidney disease (GGK/CKD), Hemodialysis (HD), and CAPD.
Use the provided context to answer the user's question accurately. If the context does not contain the answer, say that you don't know based on the provided knowledge base, but you can offer general advice.
Keep your responses helpful, supportive, and grounded in the provided facts.`
	prompt := fmt.Sprintf("%s\n\nContext information:\n%s\n\nUser Question: %s", systemInstruction, contextStr, message)

	iter := model.GenerateContentStream(ctx, genai.Text(prompt))

	// Run streaming in a background goroutine
	go func() {
		defer client.Close()
		defer close(chunkChan)
		
		var aiReplyBuilder strings.Builder

		for {
			resp, err := iter.Next()
			if err != nil {
				// We don't have a good way to return this error since the initial call succeeded.
				// We just stop streaming.
				break
			}

			for _, cand := range resp.Candidates {
				if cand.Content != nil {
					for _, part := range cand.Content.Parts {
						if txt, ok := part.(genai.Text); ok {
							textStr := string(txt)
							aiReplyBuilder.WriteString(textStr)
							chunkChan <- textStr
						}
					}
				}
			}
		}

		aiReply := strings.TrimSpace(aiReplyBuilder.String())
		if aiReply == "" {
			aiReply = "Maaf, saya tidak dapat menghasilkan respon saat ini."
			chunkChan <- aiReply
		}

		// 7. Save AI response to database
		_, dbErr := s.chatService.AddMessage(context.Background(), sessionID, "assistant", aiReply, chunks)
		if dbErr != nil {
			fmt.Printf("failed to save streaming assistant message: %v\n", dbErr)
		}
	}()

	return chunks, nil
}
