package main

import (
	"log/slog"
	"net/http"
	"os"
	"strings"
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
	logger.Info("API listening", "address", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
