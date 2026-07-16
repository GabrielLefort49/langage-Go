package enricher

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"mira/tp4/internal/core"
	"mira/tp4/internal/store/postgres"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Enricher runs note enrichment (tags, short summary, embedding) asynchronously
// on a bounded pool of workers, fed by a buffered job channel.
type Enricher struct {
	pool    *pgxpool.Pool
	store   *postgres.Store
	logger  *slog.Logger
	jobs    chan string
	workers int
	timeout time.Duration
	wg      sync.WaitGroup
}

// New starts the enricher's worker pool. Each job is processed with its own
// context bounded by timeout.
func New(pool *pgxpool.Pool, store *postgres.Store, workers int, timeout time.Duration, logger *slog.Logger) *Enricher {
	if logger == nil {
		logger = slog.Default()
	}
	e := &Enricher{pool: pool, store: store, logger: logger, jobs: make(chan string, 1000), workers: workers, timeout: timeout}
	e.wg.Add(workers)
	for i := 0; i < workers; i++ {
		go e.worker(i)
	}
	return e
}

// Publish schedules a note for enrichment. If the queue is full the job is
// dropped to keep the API responsive; the note keeps its "pending" status and
// can be retried by a future update.
func (e *Enricher) Publish(noteID string) {
	select {
	case e.jobs <- noteID:
	default:
		e.logger.Warn("enrichment queue full, dropping job", "note_id", noteID)
	}
}

// Close stops accepting new jobs and waits for in-flight work to drain, up to
// the given timeout. It is meant to be called during graceful shutdown, after
// the HTTP server has stopped accepting new requests.
func (e *Enricher) Close(timeout time.Duration) {
	close(e.jobs)
	done := make(chan struct{})
	go func() {
		e.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		e.logger.Info("enricher drained cleanly")
	case <-time.After(timeout):
		e.logger.Warn("enricher shutdown timed out, some jobs may be left pending")
	}
}

func (e *Enricher) worker(idx int) {
	defer e.wg.Done()
	for id := range e.jobs {
		ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
		if err := e.process(ctx, id); err != nil {
			e.logger.Error("enrichment failed", "note_id", id, "worker", idx, "error", err)
			if setErr := e.store.SetEnrichmentStatus(context.Background(), id, "failed"); setErr != nil {
				e.logger.Error("failed to mark note as failed", "note_id", id, "error", setErr)
			}
		} else {
			e.logger.Info("enrichment done", "note_id", id, "worker", idx)
		}
		cancel()
	}
}

func (e *Enricher) process(ctx context.Context, id string) error {
	n, err := e.store.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("fetch note: %w", err)
	}

	tags := generateTags(n.Content)
	if err := e.store.UpsertTags(ctx, id, tags); err != nil {
		return fmt.Errorf("upsert tags: %w", err)
	}

	var summary string
	if len(n.Content) > 160 {
		summary = strings.TrimSpace(n.Content[:160])
	} else {
		summary = strings.TrimSpace(n.Content)
	}
	score := float64(len(n.Content)) / 100.0
	embedding := core.BuildEmbedding(n.Content)

	if err := e.store.SetEnrichmentResult(ctx, id, &summary, &score, embedding); err != nil {
		return fmt.Errorf("store enrichment result: %w", err)
	}
	return nil
}

func generateTags(content string) []string {
	words := strings.Fields(content)
	tagsMap := map[string]struct{}{}
	for _, w := range words {
		w = strings.ToLower(strings.Trim(w, ",.;:!()[]"))
		if len(w) < 4 {
			continue
		}
		tagsMap[w] = struct{}{}
		if len(tagsMap) >= 10 {
			break
		}
	}
	tags := make([]string, 0, len(tagsMap))
	for t := range tagsMap {
		tags = append(tags, t)
	}
	return tags
}
