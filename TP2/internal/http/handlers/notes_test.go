package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"mira/tp2/internal/store"
)

func TestCreateNote(t *testing.T) {
	h := NewNotesHandler(store.NewNoteStore())
	for _, tt := range []struct {
		name, body string
		status     int
	}{
		{"success", `{"title":"Courses","content":"Acheter du lait"}`, http.StatusCreated},
		{"missing title", `{"content":"Sans titre"}`, http.StatusBadRequest},
	} {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/notes", bytes.NewBufferString(tt.body))
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != tt.status {
				t.Fatalf("status = %d, want %d: %s", rec.Code, tt.status, rec.Body.String())
			}
		})
	}
}

func TestGetMissingNote(t *testing.T) {
	h := NewNotesHandler(store.NewNoteStore())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/notes/unknown", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404: %s", rec.Code, rec.Body.String())
	}
}
