package handlers

import "mira/tp2/internal/core"

// NoteResponseDoc documents the stable success envelope in the generated spec.
type NoteResponseDoc struct {
	Data core.Note `json:"data"`
}

// NotesResponseDoc documents a paginated note list.
type NotesResponseDoc struct {
	Data []core.Note `json:"data"`
	Meta struct {
		Total  int `json:"total"`
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
	} `json:"meta"`
}

// ErrorResponseDoc documents the stable error envelope.
type ErrorResponseDoc struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}
