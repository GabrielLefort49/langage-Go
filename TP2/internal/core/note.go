package core

import "time"

// Note is the domain representation returned by the API.
type Note struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateNoteInput struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// UpdateNoteInput uses pointers so an omitted field can be distinguished from
// an explicitly supplied empty value.
type UpdateNoteInput struct {
	Title   *string `json:"title"`
	Content *string `json:"content"`
}
