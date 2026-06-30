package service

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"

	"backend/internal/domain"
	"backend/internal/repository"

	"github.com/google/uuid"
)

// RagService handles knowledge base generation and ingestion.
type RagService struct {
	chatFilePath      string
	knowledgeFilePath string
	knowledgeRepo     repository.KnowledgeRepository

	// In-memory process tracker (keyed by processID)
	processes sync.Map
}

func NewRagService(chatFilePath, knowledgeFilePath string, knowledgeRepo repository.KnowledgeRepository) *RagService {
	return &RagService{
		chatFilePath:      chatFilePath,
		knowledgeFilePath: knowledgeFilePath,
		knowledgeRepo:     knowledgeRepo,
	}
}

// chatMessage represents a single parsed WhatsApp message (header line + continuation lines).
type chatMessage struct {
	Timestamp string
	Sender    string
	Content   string // full message content (may be multiline)
}

// whatsAppHeaderRegex matches lines like:
// "12/3/24, 1:24 PM - +62 838-3104-3699: Welcome hylmi"
// The separator between time and AM/PM may be U+202F (Narrow No-Break Space), U+00A0 (NBSP), or regular space.
var whatsAppHeaderRegex = regexp.MustCompile(
	`^(\d{1,2}/\d{1,2}/\d{2,4},\s\d{1,2}:\d{2}[\s\x{202F}\x{00A0}][AP]M)\s-\s(.+?):\s(.*)$`,
)

// phoneAllMessages is the phone number whose ALL messages are included.
const phoneAllMessages = "838-3104-3699"

// phoneJustSharing is the phone number whose messages are included ONLY when they contain "just sharing".
const phoneJustSharing = "812-1515-3310"

// fileAttachedRegex matches lines like "(file attached)" indicating media-only messages.
var fileAttachedRegex = regexp.MustCompile(`\(file attached\)\s*$`)

// GenerateKnowledge reads the chat file, filters and cleans messages, then writes based-knowledge.txt.
// Returns the number of knowledge entries written.
func (s *RagService) GenerateKnowledge() (int, error) {
	messages, err := s.parseChat()
	if err != nil {
		return 0, fmt.Errorf("parse chat: %w", err)
	}

	slog.Info("parsed chat messages", "total", len(messages))

	filtered := s.filterMessages(messages)
	slog.Info("filtered knowledge messages", "count", len(filtered))

	cleaned := s.cleanMessages(filtered)
	slog.Info("cleaned knowledge entries", "count", len(cleaned))

	if err := s.writeKnowledge(cleaned); err != nil {
		return 0, fmt.Errorf("write knowledge: %w", err)
	}

	return len(cleaned), nil
}

// StartIngestion parses based-knowledge.txt, starts background ingestion into pgvector,
// and returns the process info immediately.
func (s *RagService) StartIngestion() (*domain.IngestProcess, error) {
	// Parse the knowledge file into entries
	entries, err := s.parseKnowledgeFile()
	if err != nil {
		return nil, fmt.Errorf("parse knowledge file: %w", err)
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("no entries found in knowledge file: %w", domain.ErrInvalidInput)
	}

	// Create process tracker
	processID := uuid.New().String()
	process := &domain.IngestProcess{
		ProcessID: processID,
		Status:    "pending",
		TotalRow:  len(entries),
		Ingested:  0,
	}

	// Store in memory
	s.processes.Store(processID, process)

	// Launch background goroutine
	go s.ingestInBackground(processID, entries)

	return process, nil
}

// GetIngestStatus returns the current status of an ingestion process.
func (s *RagService) GetIngestStatus(processID string) (*domain.IngestProcess, error) {
	val, ok := s.processes.Load(processID)
	if !ok {
		return nil, fmt.Errorf("process %s: %w", processID, domain.ErrNotFound)
	}

	return val.(*domain.IngestProcess), nil
}

// ingestInBackground runs in a goroutine. It clears existing chunks, inserts new ones,
// tracks progress, and removes based-knowledge.txt when finished.
func (s *RagService) ingestInBackground(processID string, entries []chatMessage) {
	val, _ := s.processes.Load(processID)
	process := val.(*domain.IngestProcess)

	// Mark as processing
	process.Status = "processing"
	s.processes.Store(processID, process)

	ctx := context.Background()

	// Clear existing knowledge before re-ingestion
	if err := s.knowledgeRepo.ClearAll(ctx); err != nil {
		slog.Error("failed to clear existing knowledge", "processId", processID, "error", err)
		process.Status = "failed"
		process.Error = err.Error()
		s.processes.Store(processID, process)
		return
	}

	slog.Info("starting knowledge ingestion", "processId", processID, "totalEntries", len(entries))

	// Use atomic counter for thread-safe progress tracking
	var ingested int64

	// Process entries concurrently with a worker pool
	const workerCount = 5
	entryCh := make(chan chatMessage, workerCount)
	var wg sync.WaitGroup
	var firstErr atomic.Value // captures first error

	// Start workers
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			
			for entry := range entryCh {
				// 1. Generate embedding using Hugging Face
				embedding, err := GenerateHFEmbedding(ctx, entry.Content)
				if err != nil {
					slog.Error("failed to embed chunk with HF", "processId", processID, "worker", workerID, "error", err)
					firstErr.CompareAndSwap(nil, err)
					continue
				}

				// 2. Insert into Qdrant
				_, err = s.knowledgeRepo.InsertChunk(ctx, entry.Content, entry.Timestamp, embedding)
				if err != nil {
					slog.Error("failed to insert chunk",
						"processId", processID,
						"worker", workerID,
						"error", err,
					)
					firstErr.CompareAndSwap(nil, err)
					continue
				}
				count := atomic.AddInt64(&ingested, 1)
				// Update process progress periodically (every 10 or at the end)
				if count%10 == 0 || int(count) == len(entries) {
					process.Ingested = int(count)
					s.processes.Store(processID, process)
				}
			}
		}(i)
	}

	// Feed entries to workers
	for _, entry := range entries {
		entryCh <- entry
	}
	close(entryCh)

	// Wait for all workers to complete
	wg.Wait()

	// Final update
	finalCount := int(atomic.LoadInt64(&ingested))
	process.Ingested = finalCount

	if errVal := firstErr.Load(); errVal != nil {
		process.Status = "failed"
		process.Error = errVal.(error).Error()
		s.processes.Store(processID, process)
		slog.Error("ingestion completed with errors",
			"processId", processID,
			"ingested", finalCount,
			"total", len(entries),
		)
		return
	}

	// Success — remove based-knowledge.txt
	if err := os.Remove(s.knowledgeFilePath); err != nil {
		slog.Warn("failed to remove knowledge file after ingestion",
			"processId", processID,
			"path", s.knowledgeFilePath,
			"error", err,
		)
		// Not fatal — ingestion still succeeded
	}

	process.Status = "finished"
	s.processes.Store(processID, process)

	slog.Info("knowledge ingestion completed",
		"processId", processID,
		"ingested", finalCount,
	)
}

// parseKnowledgeFile reads based-knowledge.txt and splits it into entries.
// Format: entries separated by "---", each starting with [timestamp].
func (s *RagService) parseKnowledgeFile() ([]chatMessage, error) {
	file, err := os.Open(s.knowledgeFilePath)
	if err != nil {
		return nil, fmt.Errorf("open knowledge file: %w", err)
	}
	defer file.Close()

	var entries []chatMessage
	var currentLines []string
	var currentTimestamp string

	timestampRegex := regexp.MustCompile(`^\[(.+)\]$`)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		// Entry separator
		if strings.TrimSpace(line) == "---" {
			if len(currentLines) > 0 {
				content := strings.TrimSpace(strings.Join(currentLines, "\n"))
				if content != "" {
					entries = append(entries, chatMessage{
						Timestamp: currentTimestamp,
						Content:   content,
					})
				}
			}
			currentLines = nil
			currentTimestamp = ""
			continue
		}

		// Timestamp header
		if matches := timestampRegex.FindStringSubmatch(strings.TrimSpace(line)); matches != nil {
			currentTimestamp = matches[1]
			continue
		}

		currentLines = append(currentLines, line)
	}

	// Don't forget the last entry
	if len(currentLines) > 0 {
		content := strings.TrimSpace(strings.Join(currentLines, "\n"))
		if content != "" {
			entries = append(entries, chatMessage{
				Timestamp: currentTimestamp,
				Content:   content,
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan knowledge file: %w", err)
	}

	return entries, nil
}

// parseChat reads the WhatsApp chat export and groups header lines with their continuation lines.
func (s *RagService) parseChat() ([]chatMessage, error) {
	file, err := os.Open(s.chatFilePath)
	if err != nil {
		return nil, fmt.Errorf("open chat file: %w", err)
	}
	defer file.Close()

	var messages []chatMessage
	var current *chatMessage

	scanner := bufio.NewScanner(file)
	// Increase buffer for long lines
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		if matches := whatsAppHeaderRegex.FindStringSubmatch(line); matches != nil {
			// Save previous message
			if current != nil {
				messages = append(messages, *current)
			}
			current = &chatMessage{
				Timestamp: matches[1],
				Sender:    matches[2],
				Content:   matches[3],
			}
		} else if current != nil {
			// Continuation line — append to current message
			current.Content += "\n" + line
		}
		// Lines before any header are ignored (e.g., encryption notice)
	}

	// Don't forget the last message
	if current != nil {
		messages = append(messages, *current)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan chat file: %w", err)
	}

	return messages, nil
}

// filterMessages selects messages based on the phone number rules:
// - phoneAllMessages: include all messages
// - phoneJustSharing: include only messages containing "just sharing" (case-insensitive)
func (s *RagService) filterMessages(messages []chatMessage) []chatMessage {
	var filtered []chatMessage

	for _, msg := range messages {
		sender := msg.Sender

		if strings.Contains(sender, phoneAllMessages) {
			filtered = append(filtered, msg)
		} else if strings.Contains(sender, phoneJustSharing) {
			contentLower := strings.ToLower(msg.Content)
			if strings.Contains(contentLower, "just sharing") {
				filtered = append(filtered, msg)
			}
		}
	}

	return filtered
}

// cleanMessages removes noise from filtered messages:
// - Strips file attachment lines
// - Removes URLs/links
// - Removes WhatsApp formatting markers (*bold*, _italic_)
// - Trims excessive whitespace
// - Skips messages that become empty after cleaning
func (s *RagService) cleanMessages(messages []chatMessage) []chatMessage {
	urlRegex := regexp.MustCompile(`https?://\S+`)
	mentionRegex := regexp.MustCompile(`@\d+`)
	editedRegex := regexp.MustCompile(`<This message was edited>`)
	waFormatRegex := regexp.MustCompile(`\*([^*]+)\*`)

	var cleaned []chatMessage

	for _, msg := range messages {
		content := msg.Content

		// Skip file-only messages
		if fileAttachedRegex.MatchString(strings.TrimSpace(content)) {
			continue
		}

		// Remove URLs
		content = urlRegex.ReplaceAllString(content, "")

		// Remove @mentions
		content = mentionRegex.ReplaceAllString(content, "")

		// Remove "edited" markers
		content = editedRegex.ReplaceAllString(content, "")

		// Remove WhatsApp bold markers but keep the text inside
		content = waFormatRegex.ReplaceAllString(content, "$1")

		// Clean up lines: trim each line, remove empty lines from attachment references
		lines := strings.Split(content, "\n")
		var cleanedLines []string
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			// Skip lines that are just file references
			if fileAttachedRegex.MatchString(trimmed) {
				continue
			}
			cleanedLines = append(cleanedLines, trimmed)
		}

		content = strings.Join(cleanedLines, "\n")

		// Collapse multiple blank lines into one
		multiNewline := regexp.MustCompile(`\n{3,}`)
		content = multiNewline.ReplaceAllString(content, "\n\n")

		content = strings.TrimSpace(content)

		// Skip if empty after cleaning
		if content == "" {
			continue
		}

		// Skip short messages (less than 100 chars) — not substantive enough for knowledge base
		if len([]rune(content)) < 100 {
			continue
		}

		cleaned = append(cleaned, chatMessage{
			Timestamp: msg.Timestamp,
			Sender:    msg.Sender,
			Content:   content,
		})
	}

	return cleaned
}

// writeKnowledge writes the cleaned messages to based-knowledge.txt in a structured format.
func (s *RagService) writeKnowledge(messages []chatMessage) error {
	file, err := os.Create(s.knowledgeFilePath)
	if err != nil {
		return fmt.Errorf("create knowledge file: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	for i, msg := range messages {
		// Write separator between entries
		if i > 0 {
			fmt.Fprintln(writer, "")
			fmt.Fprintln(writer, "---")
			fmt.Fprintln(writer, "")
		}

		fmt.Fprintf(writer, "[%s]\n", msg.Timestamp)
		fmt.Fprintln(writer, msg.Content)
	}

	return nil
}
