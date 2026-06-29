package handler

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type RouterOptions struct {
	UserHandler   *UserHandler
	ChatHandler   *ChatHandler
	AuthHandler   *AuthHandler
	WeightHandler *WeightHandler
	LabHandler    *LabHandler
}

type contextKey string
const userIDKey contextKey = "userID"

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"duration", time.Since(start).String(),
		)
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func jwtAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		secret := os.Getenv("JWT_SECRET")
		if secret == "" {
			secret = "default_secret_for_dev_only"
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			userID := claims["sub"].(string)
			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}
}

func SetupRouter(opts RouterOptions) *http.ServeMux {
	mux := http.NewServeMux()

	// Auth
	mux.HandleFunc("POST /api/auth/register", opts.AuthHandler.Register)
	mux.HandleFunc("POST /api/auth/login", opts.AuthHandler.Login)

	// Users - old register left intact for now, but we'll protect Get and Update
	mux.HandleFunc("POST /api/users", opts.UserHandler.Register)
	mux.HandleFunc("GET /api/users/{id}", jwtAuthMiddleware(opts.UserHandler.Get))
	mux.HandleFunc("PUT /api/users/{id}", jwtAuthMiddleware(opts.UserHandler.Update))

	// Weights
	mux.HandleFunc("POST /api/users/{id}/weights", jwtAuthMiddleware(opts.WeightHandler.Create))
	mux.HandleFunc("GET /api/users/{id}/weights", jwtAuthMiddleware(opts.WeightHandler.List))
	mux.HandleFunc("PUT /api/weights/{id}", jwtAuthMiddleware(opts.WeightHandler.Update))
	mux.HandleFunc("DELETE /api/weights/{id}", jwtAuthMiddleware(opts.WeightHandler.Delete))

	// Labs
	mux.HandleFunc("POST /api/users/{id}/labs", jwtAuthMiddleware(opts.LabHandler.Create))
	mux.HandleFunc("GET /api/users/{id}/labs", jwtAuthMiddleware(opts.LabHandler.List))
	mux.HandleFunc("PUT /api/labs/{id}", jwtAuthMiddleware(opts.LabHandler.Update))
	mux.HandleFunc("DELETE /api/labs/{id}", jwtAuthMiddleware(opts.LabHandler.Delete))

	// Chat Sessions
	mux.HandleFunc("POST /api/sessions", jwtAuthMiddleware(opts.ChatHandler.CreateSession))
	mux.HandleFunc("GET /api/users/{userId}/sessions", jwtAuthMiddleware(opts.ChatHandler.ListSessions))
	mux.HandleFunc("DELETE /api/sessions/{id}", jwtAuthMiddleware(opts.ChatHandler.DeleteSession))

	// Messages
	mux.HandleFunc("POST /api/sessions/{id}/messages", jwtAuthMiddleware(opts.ChatHandler.AddMessage))
	mux.HandleFunc("GET /api/sessions/{id}/messages", jwtAuthMiddleware(opts.ChatHandler.ListMessages))

	// Apply global middlewares by wrapping the mux
	wrapper := http.NewServeMux()
	wrapper.Handle("/", corsMiddleware(loggingMiddleware(mux)))

	return wrapper
}
