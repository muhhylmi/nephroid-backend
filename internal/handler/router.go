package handler

import (
	"log/slog"
	"net/http"
	"time"
)

type RouterOptions struct {
	UserHandler *UserHandler
	ChatHandler *ChatHandler
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Note: A real app would use a custom response writer to capture status code,
		// but keeping this simple for the baseline CRUD setup.
		next.ServeHTTP(w, r)
		
		slog.Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"duration", time.Since(start).String(),
		)
	})
}

func SetupRouter(opts RouterOptions) *http.ServeMux {
	mux := http.NewServeMux()

	// Users
	mux.HandleFunc("POST /api/users", opts.UserHandler.Register)
	mux.HandleFunc("GET /api/users/{id}", opts.UserHandler.Get)

	// Chat Sessions
	mux.HandleFunc("POST /api/sessions", opts.ChatHandler.CreateSession)
	mux.HandleFunc("GET /api/users/{userId}/sessions", opts.ChatHandler.ListSessions)
	mux.HandleFunc("DELETE /api/sessions/{id}", opts.ChatHandler.DeleteSession)

	// Messages
	mux.HandleFunc("POST /api/sessions/{id}/messages", opts.ChatHandler.AddMessage)
	mux.HandleFunc("GET /api/sessions/{id}/messages", opts.ChatHandler.ListMessages)

	// Apply global middleware by wrapping the mux
	wrapper := http.NewServeMux()
	wrapper.Handle("/", loggingMiddleware(mux))

	return wrapper
}
