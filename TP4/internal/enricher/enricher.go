package enricher

import (
	"context"
	"strings"
	"time"

	"mira/tp4/internal/core"
	"mira/tp4/internal/store/postgres"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Enricher struct {
	pool    *pgxpool.Pool
	store   *postgres.Store
	jobs    chan string
	workers int
	timeout time.Duration
}

func New(pool *pgxpool.Pool, store *postgres.Store, workers int, timeout time.Duration) *Enricher {
	e := &Enricher{pool: pool, store: store, jobs: make(chan string, 1000), workers: workers, timeout: timeout}
	for i := 0; i < workers; i++ {
		go e.worker(i)
	}
	return e
}

func (e *Enricher) Publish(noteID string) {
	select {
	case e.jobs <- noteID:
	default:
		// drop if queue full to keep API fast; could also block or log
	}
}

func (e *Enricher) worker(idx int) {
	for id := range e.jobs {
		ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
		_ = e.process(ctx, id)
		cancel()
	}
}

func (e *Enricher) process(ctx context.Context, id string) error {
	// fetch note
	n, err := e.store.Get(ctx, id)
	if err != nil {
		return err
	}
	// generate tags: simple split of content
	tags := generateTags(n.Content)
	_ = e.store.UpsertTags(ctx, id, tags)
	// short summary: first 160 chars
	var summary string
	if len(n.Content) > 160 {
		summary = strings.TrimSpace(n.Content[:160])
	} else {
		summary = strings.TrimSpace(n.Content)
	}
	// score: simple score based on length
	score := float64(len(n.Content)) / 100.0
	// embedding: deterministic pseudo-embedding derived from content
	embedding := core.BuildEmbedding(n.Content)
	// store results
	// convert []float32 to []float32 as pgx accepts
	_ = e.store.SetEnrichmentResult(ctx, id, &summary, &score, embedding)
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
