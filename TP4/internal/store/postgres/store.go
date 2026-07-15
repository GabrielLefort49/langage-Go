package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"mira/tp4/internal/core"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

func (s *Store) Create(ctx context.Context, input core.CreateNoteInput) (core.Note, error) {
	id := fmt.Sprintf("note-%d", time.Now().UnixNano())
	now := time.Now().UTC()
	_, err := s.pool.Exec(ctx, `INSERT INTO notes (id,title,content,created_at,updated_at,enrichment_status) VALUES ($1,$2,$3,$4,$5,'pending')`, id, input.Title, input.Content, now, now)
	if err != nil {
		return core.Note{}, err
	}
	return core.Note{ID: id, Title: input.Title, Content: input.Content, CreatedAt: now, UpdatedAt: now, EnrichmentStatus: "pending"}, nil
}

func (s *Store) Get(ctx context.Context, id string) (core.Note, error) {
	row := s.pool.QueryRow(ctx, `SELECT id,title,content,created_at,updated_at,enrichment_status,short_summary,score FROM notes WHERE id=$1`, id)
	var n core.Note
	var short *string
	var score *float64
	if err := row.Scan(&n.ID, &n.Title, &n.Content, &n.CreatedAt, &n.UpdatedAt, &n.EnrichmentStatus, &short, &score); err != nil {
		return core.Note{}, err
	}
	n.ShortSummary = short
	n.Score = score
	// load tags
	rows, err := s.pool.Query(ctx, `SELECT tag FROM note_tags WHERE note_id=$1`, id)
	if err == nil {
		defer rows.Close()
		tags := []string{}
		for rows.Next() {
			var t string
			_ = rows.Scan(&t)
			tags = append(tags, t)
		}
		n.Tags = tags
	}
	return n, nil
}

func (s *Store) List(ctx context.Context, limit, offset int) ([]core.Note, int, error) {
	rows, err := s.pool.Query(ctx, `SELECT id,title,content,created_at,updated_at,enrichment_status FROM notes ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	notes := []core.Note{}
	for rows.Next() {
		var n core.Note
		_ = rows.Scan(&n.ID, &n.Title, &n.Content, &n.CreatedAt, &n.UpdatedAt, &n.EnrichmentStatus)
		notes = append(notes, n)
	}
	var total int
	_ = s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM notes`).Scan(&total)
	return notes, total, nil
}

func (s *Store) Search(ctx context.Context, query string, limit, offset int) ([]core.Note, int, error) {
	// hybrid: full-text match OR vector similarity if query is short; for simplicity use full-text
	q := strings.TrimSpace(query)
	rows, err := s.pool.Query(ctx, `SELECT id,title,content,created_at,updated_at,enrichment_status FROM notes WHERE to_tsvector('english', coalesce(title,'')||' '||coalesce(content,'')) @@ plainto_tsquery('english',$1) ORDER BY created_at DESC LIMIT $2 OFFSET $3`, q, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	notes := []core.Note{}
	for rows.Next() {
		var n core.Note
		_ = rows.Scan(&n.ID, &n.Title, &n.Content, &n.CreatedAt, &n.UpdatedAt, &n.EnrichmentStatus)
		notes = append(notes, n)
	}
	var total int
	_ = s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM notes WHERE to_tsvector('english', coalesce(title,'')||' '||coalesce(content,'')) @@ plainto_tsquery('english',$1)`, q).Scan(&total)
	return notes, total, nil
}

func (s *Store) Update(ctx context.Context, id string, input core.UpdateNoteInput) (core.Note, error) {
	// simple partial update
	now := time.Now().UTC()
	if input.Title != nil && input.Content != nil {
		_, err := s.pool.Exec(ctx, `UPDATE notes SET title=$1, content=$2, updated_at=$3, enrichment_status='pending' WHERE id=$4`, *input.Title, *input.Content, now, id)
		if err != nil {
			return core.Note{}, err
		}
	} else if input.Title != nil {
		_, err := s.pool.Exec(ctx, `UPDATE notes SET title=$1, updated_at=$2, enrichment_status='pending' WHERE id=$3`, *input.Title, now, id)
		if err != nil {
			return core.Note{}, err
		}
	} else if input.Content != nil {
		_, err := s.pool.Exec(ctx, `UPDATE notes SET content=$1, updated_at=$2, enrichment_status='pending' WHERE id=$3`, *input.Content, now, id)
		if err != nil {
			return core.Note{}, err
		}
	}
	return s.Get(ctx, id)
}

func (s *Store) Delete(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM notes WHERE id=$1`, id)
	return err
}

// Helpers used by enricher to set metadata atomically
func (s *Store) UpsertTags(ctx context.Context, id string, tags []string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = tx.Exec(ctx, `DELETE FROM note_tags WHERE note_id=$1`, id)
	if err != nil {
		return err
	}
	for _, t := range tags {
		if strings.TrimSpace(t) == "" {
			continue
		}
		_, err = tx.Exec(ctx, `INSERT INTO note_tags (note_id,tag) VALUES ($1,$2)`, id, t)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) SetEnrichmentStatus(ctx context.Context, id, status string) error {
	_, err := s.pool.Exec(ctx, `UPDATE notes SET enrichment_status=$1 WHERE id=$2`, status, id)
	return err
}

func (s *Store) SetEnrichmentResult(ctx context.Context, id string, summary *string, score *float64, embedding []float32) error {
	// update note and embedding table
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = tx.Exec(ctx, `UPDATE notes SET short_summary=$1, score=$2, enrichment_status='done' WHERE id=$3`, summary, score, id)
	if err != nil {
		return err
	}
	if embedding != nil {
		payload, err := json.Marshal(embedding)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO note_embeddings (note_id, embedding) VALUES ($1, $2) ON CONFLICT (note_id) DO UPDATE SET embedding = EXCLUDED.embedding`, id, string(payload))
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
