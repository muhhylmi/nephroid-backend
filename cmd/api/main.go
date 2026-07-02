package main

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"backend/internal/handler"
	"backend/internal/repository"
	"backend/internal/server"
	"backend/internal/service"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/qdrant/go-client/qdrant"
)

func main() {
	// 1. Setup Structured Logging (slog)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Load .env if it exists
	if err := godotenv.Load(); err != nil {
		slog.Info("No .env file found, relying on environment variables")
	}

	// 2. Initialize Dependencies
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		slog.Error("DATABASE_URL is not set in environment")
		os.Exit(1)
	}

	dbPool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		slog.Error("Unable to create connection pool", "error", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	// Verify DB connection
	if err := dbPool.Ping(context.Background()); err != nil {
		slog.Error("Database ping failed", "error", err)
		os.Exit(1)
	}

	// Connect to Qdrant (supports local and qdrant.io cloud)
	qdrantHost := os.Getenv("QDRANT_HOST")
	if qdrantHost == "" {
		qdrantHost = "localhost"
	}
	// Clean up host if user accidentally provided URL with scheme or port
	qdrantHost = strings.TrimPrefix(qdrantHost, "https://")
	qdrantHost = strings.TrimPrefix(qdrantHost, "http://")
	if idx := strings.Index(qdrantHost, ":"); idx != -1 {
		qdrantHost = qdrantHost[:idx]
	}
	
	qdrantPortStr := os.Getenv("QDRANT_PORT")
	qdrantPort := 6334
	if qdrantPortStr != "" {
		if p, err := strconv.Atoi(qdrantPortStr); err == nil {
			qdrantPort = p
		}
	}

	qdrantAPIKey := os.Getenv("QDRANT_API_KEY")
	useTLS := os.Getenv("QDRANT_USE_TLS") == "true" || qdrantAPIKey != "" // Cloud typically uses TLS and API Key

	qdrantClient, err := qdrant.NewClient(&qdrant.Config{
		Host:   qdrantHost,
		Port:   qdrantPort,
		APIKey: qdrantAPIKey,
		UseTLS: useTLS,
	})
	if err != nil {
		slog.Error("Unable to create Qdrant client", "error", err)
		os.Exit(1)
	}
	defer qdrantClient.Close()

	// 3. Initialize Repositories
	userRepo := repository.NewUserRepository(dbPool)
	sessionRepo := repository.NewSessionRepository(dbPool)
	messageRepo := repository.NewMessageRepository(dbPool)
	weightRepo := repository.NewWeightRepository(dbPool)
	labRepo := repository.NewLabRepository(dbPool)
	knowledgeRepo := repository.NewKnowledgeRepository(qdrantClient)

	// Ensure Qdrant collection exists
	if err := knowledgeRepo.EnsureCollectionExists(context.Background()); err != nil {
		slog.Error("Failed to ensure Qdrant collection", "error", err)
		os.Exit(1)
	}

	// 4. Initialize Services
	userService := service.NewUserService(userRepo)
	chatService := service.NewChatService(sessionRepo, messageRepo)
	authService := service.NewAuthService(userRepo)
	ragService := service.NewRagService("../chat.txt", "../based-knowledge.txt", knowledgeRepo)
	aiService := service.NewAiService(knowledgeRepo, chatService)

	// 5. Initialize Handlers
	userHandler := handler.NewUserHandler(userService)
	chatHandler := handler.NewChatHandler(chatService)
	authHandler := handler.NewAuthHandler(authService)
	weightHandler := handler.NewWeightHandler(weightRepo)
	labHandler := handler.NewLabHandler(labRepo)
	ragHandler := handler.NewRagHandler(ragService, aiService)

	// 6. Setup Router
	router := handler.SetupRouter(handler.RouterOptions{
		UserHandler:   userHandler,
		ChatHandler:   chatHandler,
		AuthHandler:   authHandler,
		WeightHandler: weightHandler,
		LabHandler:    labHandler,
		RagHandler:    ragHandler,
	})

	// 7. Configure and Start Server using Functional Options
	port := os.Getenv("PORT")
	if port == "" {
		slog.Error("PORT is not set in environment")
		os.Exit(1)
	}

	srv := server.New(
		server.WithAddr(":"+port),
		server.WithHandler(router),
	)

	slog.Info("Starting nephroid backend...")
	if err := srv.Start(); err != nil {
		slog.Error("Server stopped", "error", err)
		os.Exit(1)
	}
}
