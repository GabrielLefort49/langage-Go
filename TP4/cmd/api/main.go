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

	"mira/tp4/internal/enricher"
	"mira/tp4/internal/http/handlers"
	pgstore "mira/tp4/internal/store/postgres"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/mira?sslmode=disable"
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		logger.Error("pgxpool.New", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := applyMigrations(ctx, pool); err != nil {
		logger.Error("migrations", "error", err)
		os.Exit(1)
	}

	store := pgstore.New(pool)
	enr := enricher.New(pool, store, 4, 15*time.Second, logger)

	notesHandler := handlers.NewNotesHandler(store, enr)
	searchHandler := handlers.NewSearchHandler(store)

	mux := http.NewServeMux()
	mux.Handle("/healthz", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	mux.Handle("/swagger", http.HandlerFunc(handlers.SwaggerUIHandler))
	mux.Handle("/swagger.json", http.HandlerFunc(handlers.SwaggerSpecHandler))
	mux.Handle("/api/v1/search", searchHandler)
	mux.Handle("/api/v1/notes", notesHandler)
	mux.Handle("/api/v1/notes/", notesHandler)

	addr := os.Getenv("PORT")
	if addr == "" {
		addr = "8080"
	}
	if !strings.HasPrefix(addr, ":") {
		addr = ":" + addr
	}

	// Wrap with CORS handler to allow Swagger aggregator and cross-origin requests.
	handler := cors(mux)
	srv := &http.Server{Addr: addr, Handler: handler, ReadHeaderTimeout: 5 * time.Second}

	shutdownCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("API listening", "address", addr)
		serverErr <- srv.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			if errors.Is(err, syscall.EADDRINUSE) {
				logger.Error("port already in use, stop the previous process or set PORT to another value", "address", addr)
			} else {
				logger.Error("server stopped unexpectedly", "error", err)
			}
			os.Exit(1)
		}
	case <-shutdownCtx.Done():
		stop()
		logger.Info("shutdown signal received, draining connections")
		httpShutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(httpShutdownCtx); err != nil {
			logger.Error("graceful shutdown failed", "error", err)
		}
		enr.Close(10 * time.Second)
		logger.Info("server stopped")
	}
}

// cors returns a handler that sets permissive CORS headers and handles OPTIONS preflight.
func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func applyMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	b, err := os.ReadFile("migrations/001_init.sql")
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, string(b))
	return err
}
