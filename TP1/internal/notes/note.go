package notes

import "time"

type Note struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

func NewNote(id int, title, content string) Note {
	return Note{
		ID:        id,
		Title:     title,
		Content:   content,
		CreatedAt: time.Now(),
	}
}
