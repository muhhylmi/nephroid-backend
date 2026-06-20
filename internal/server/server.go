package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Server struct {
	httpServer *http.Server
}

type Option func(*http.Server)

func WithAddr(addr string) Option {
	return func(s *http.Server) {
		s.Addr = addr
	}
}

func WithHandler(handler http.Handler) Option {
	return func(s *http.Server) {
		s.Handler = handler
	}
}

func WithTimeouts(read, write, idle time.Duration) Option {
	return func(s *http.Server) {
		s.ReadTimeout = read
		s.WriteTimeout = write
		s.IdleTimeout = idle
	}
}

func New(opts ...Option) *Server {
	// Default options
	srv := &http.Server{
		Addr:         ":8080",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	for _, opt := range opts {
		opt(srv)
	}

	return &Server{
		httpServer: srv,
	}
}

func (s *Server) Start() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		slog.Info(fmt.Sprintf("Server is listening on %s", s.httpServer.Addr))
		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("server error: %w", err)
	case <-ctx.Done():
		slog.Info("Shutting down gracefully...")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("server shutdown error: %w", err)
		}
		slog.Info("Server stopped")
	}

	return nil
}
