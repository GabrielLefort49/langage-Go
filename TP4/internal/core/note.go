package core

import "time"

type Note struct {
	ID               string    `json:"id"`
	Title            string    `json:"title"`
	Content          string    `json:"content"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	EnrichmentStatus string    `json:"enrichment_status"`
	ShortSummary     *string   `json:"short_summary,omitempty"`
	Score            *float64  `json:"score,omitempty"`
	Tags             []string  `json:"tags,omitempty"`
}

type CreateNoteInput struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

type UpdateNoteInput struct {
	Title   *string `json:"title,omitempty"`
	Content *string `json:"content,omitempty"`
}
