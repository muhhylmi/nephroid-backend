package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// GenerateHFEmbedding calls the Hugging Face Inference API to generate an embedding for the given text.
func GenerateHFEmbedding(ctx context.Context, text string) ([]float32, error) {
	apiKey := os.Getenv("HF_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("HF_API_KEY is not set")
	}

	modelURL := os.Getenv("HF_EMBEDDING_MODEL_URL")
	if modelURL == "" {
		// Default to a small, fast model (384 dimensions)
		modelURL = "https://api-inference.huggingface.co/pipeline/feature-extraction/sentence-transformers/all-MiniLM-L6-v2"
	}

	reqBody, err := json.Marshal(map[string]interface{}{
		"inputs": text,
		"options": map[string]interface{}{
			"wait_for_model": true,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", modelURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("hf api error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	// HF feature-extraction returns a JSON array of floats for single input
	var embedding []float32
	if err := json.NewDecoder(resp.Body).Decode(&embedding); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(embedding) == 0 {
		return nil, fmt.Errorf("empty embedding returned")
	}

	return embedding, nil
}
