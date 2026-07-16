package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"mira/tp2/internal/http/handlers"
	"mira/tp2/internal/http/middleware"
	"mira/tp2/internal/http/openapi"
	"mira/tp2/internal/store"
)

// @title Mira API
// @version 1.0
// @description API HTTP de gestion de notes.
// @host localhost:8080
// @BasePath /
func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	notes := handlers.NewNotesHandler(store.NewNoteStore())
	mux := http.NewServeMux()
	mux.Handle("/api/v1/notes", notes)
	mux.Handle("/api/v1/notes/", notes)
	mux.HandleFunc("/api/v1/search", notes.Search)
	mux.HandleFunc("/openapi.yaml", openapi.Spec)
	mux.HandleFunc("/swagger-generated.yaml", openapi.GeneratedSpec)
	mux.HandleFunc("/swagger/", openapi.UI)

	var handler http.Handler = mux
	// CORS should be applied early so preflight requests are handled.
	handler = middleware.CORS(handler)
	handler = middleware.RequestID(handler)
	handler = middleware.Logging(logger)(handler)
	handler = middleware.Recovery(logger)(handler)
	handler = middleware.Timeout(5 * time.Second)(handler)

	addr := os.Getenv("PORT")
	if addr == "" {
		addr = "8080"
	}
	if !strings.HasPrefix(addr, ":") {
		addr = ":" + addr
	}
	server := &http.Server{Addr: addr, Handler: handler, ReadHeaderTimeout: 5 * time.Second}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("API listening", "address", server.Addr)
		serverErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server stopped unexpectedly", "error", err)
			os.Exit(1)
		}
	case <-ctx.Done():
		stop()
		logger.Info("shutdown signal received, draining connections")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("graceful shutdown failed", "error", err)
			os.Exit(1)
		}
		logger.Info("server stopped")
	}
}
