package search

import (
	"strings"

	"mira/internal/notes"
)

func Search(list []notes.Note, query string) []notes.Note {

	query = strings.ToLower(query)

	var result []notes.Note

	for _, note := range list {

		text := strings.ToLower(note.Title + " " + note.Content)

		if strings.Contains(text, query) {
			result = append(result, note)
		}
	}

	return result
}
