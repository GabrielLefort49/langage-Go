package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"mira/tp4/internal/core"
	"mira/tp4/internal/enricher"
	handlers "mira/tp4/internal/http/handlers"
	pgstore "mira/tp4/internal/store/postgres"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestNotesAPIEndToEnd(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/mira?sslmode=disable"
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	defer pool.Close()

	if err := applyTestMigrations(ctx, pool); err != nil {
		t.Fatalf("applyTestMigrations: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM note_tags; DELETE FROM note_embeddings; DELETE FROM notes;`); err != nil {
		t.Fatalf("truncate tables: %v", err)
	}

	store := pgstore.New(pool)
	enr := enricher.New(pool, store, 1, 2*time.Second)
	h := handlers.NewNotesHandler(store, enr)
	search := handlers.NewSearchHandler(store)

	mux := http.NewServeMux()
	mux.Handle("/healthz", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	mux.Handle("/swagger.json", http.HandlerFunc(handlers.SwaggerSpecHandler))
	mux.Handle("/api/v1/search", search)
	mux.Handle("/api/v1/notes", h)
	mux.Handle("/api/v1/notes/", h)

	server := httptest.NewServer(mux)
	defer server.Close()

	createReq := map[string]string{"title": "TP4 search test", "content": "This note contains important keywords for hybrid search and enrichment"}
	payload, _ := json.Marshal(createReq)
	resp, err := http.Post(server.URL+"/api/v1/notes", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("create note: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create note status = %d", resp.StatusCode)
	}

	var created struct {
		Data core.Note `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	deadline := time.Now().Add(8 * time.Second)
	var note core.Note
	for time.Now().Before(deadline) {
		noteResp, err := http.Get(server.URL + "/api/v1/notes/" + created.Data.ID)
		if err != nil {
			t.Fatalf("get note: %v", err)
		}
		var got struct {
			Data core.Note `json:"data"`
		}
		if err := json.NewDecoder(noteResp.Body).Decode(&got); err != nil {
			noteResp.Body.Close()
			t.Fatalf("decode get note: %v", err)
		}
		noteResp.Body.Close()
		note = got.Data
		if note.EnrichmentStatus == "done" {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	if note.EnrichmentStatus != "done" {
		t.Fatalf("expected enrichment status done, got %q", note.EnrichmentStatus)
	}
	if note.ShortSummary == nil || *note.ShortSummary == "" {
		t.Fatalf("expected non-empty short summary")
	}
	if note.Score == nil || *note.Score <= 0 {
		t.Fatalf("expected positive score")
	}
	if len(note.Tags) == 0 {
		t.Fatalf("expected at least one tag")
	}

	healthResp, err := http.Get(server.URL + "/healthz")
	if err != nil {
		t.Fatalf("healthz: %v", err)
	}
	defer healthResp.Body.Close()
	if healthResp.StatusCode != http.StatusOK {
		t.Fatalf("healthz status = %d", healthResp.StatusCode)
	}

	swaggerResp, err := http.Get(server.URL + "/swagger.json")
	if err != nil {
		t.Fatalf("swagger: %v", err)
	}
	defer swaggerResp.Body.Close()
	if swaggerResp.StatusCode != http.StatusOK {
		t.Fatalf("swagger status = %d", swaggerResp.StatusCode)
	}

	searchResp, err := http.Get(server.URL + "/api/v1/search?q=keywords")
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	defer searchResp.Body.Close()
	if searchResp.StatusCode != http.StatusOK {
		t.Fatalf("search status = %d", searchResp.StatusCode)
	}

	var searchResult struct {
		Data []core.Note `json:"data"`
	}
	if err := json.NewDecoder(searchResp.Body).Decode(&searchResult); err != nil {
		t.Fatalf("decode search response: %v", err)
	}
	if len(searchResult.Data) == 0 {
		t.Fatalf("expected at least one search result")
	}

	variantResp, err := http.Get(server.URL + "/api/v1/search?q=keyword")
	if err != nil {
		t.Fatalf("variant search: %v", err)
	}
	defer variantResp.Body.Close()
	if variantResp.StatusCode != http.StatusOK {
		t.Fatalf("variant search status = %d", variantResp.StatusCode)
	}

	var variantResult struct {
		Data []core.Note `json:"data"`
	}
	if err := json.NewDecoder(variantResp.Body).Decode(&variantResult); err != nil {
		t.Fatalf("decode variant search response: %v", err)
	}
	if len(variantResult.Data) == 0 {
		t.Fatalf("expected vector-style search to match near variants")
	}

	fmt.Println("integration test passed for note", created.Data.ID)
}

func applyTestMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	path := filepath.Join("..", "..", "..", "migrations", "001_init.sql")
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, string(content))
	return err
}
