package store

import (
	"errors"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"mira/tp2/internal/core"
)

var ErrNotFound = errors.New("note not found")

// NoteStore is an in-memory, concurrency-safe note repository.
type NoteStore struct {
	mu     sync.RWMutex
	notes  map[string]core.Note
	nextID uint64
}

func NewNoteStore() *NoteStore { return &NoteStore{notes: make(map[string]core.Note)} }

func (s *NoteStore) Create(input core.CreateNoteInput) core.Note {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	now := time.Now().UTC()
	note := core.Note{ID: formatID(s.nextID), Title: input.Title, Content: input.Content, CreatedAt: now, UpdatedAt: now}
	s.notes[note.ID] = note
	return note
}

func (s *NoteStore) Get(id string) (core.Note, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	note, ok := s.notes[id]
	if !ok {
		return core.Note{}, ErrNotFound
	}
	return note, nil
}

func (s *NoteStore) List(limit, offset int) ([]core.Note, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	notes := make([]core.Note, 0, len(s.notes))
	for _, note := range s.notes {
		notes = append(notes, note)
	}
	sort.Slice(notes, func(i, j int) bool { return notes[i].CreatedAt.After(notes[j].CreatedAt) })
	total := len(notes)
	if offset >= total {
		return []core.Note{}, total
	}
	end := total
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}
	return notes[offset:end], total
}

func (s *NoteStore) Search(query string, limit, offset int) ([]core.Note, int) {
	query = strings.ToLower(query)
	s.mu.RLock()
	defer s.mu.RUnlock()
	notes := make([]core.Note, 0)
	for _, note := range s.notes {
		if strings.Contains(strings.ToLower(note.Title), query) || strings.Contains(strings.ToLower(note.Content), query) {
			notes = append(notes, note)
		}
	}
	sort.Slice(notes, func(i, j int) bool { return notes[i].CreatedAt.After(notes[j].CreatedAt) })
	total := len(notes)
	if offset >= total {
		return []core.Note{}, total
	}
	end := total
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}
	return notes[offset:end], total
}

func (s *NoteStore) Update(id string, input core.UpdateNoteInput) (core.Note, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	note, ok := s.notes[id]
	if !ok {
		return core.Note{}, ErrNotFound
	}
	if input.Title != nil {
		note.Title = *input.Title
	}
	if input.Content != nil {
		note.Content = *input.Content
	}
	note.UpdatedAt = time.Now().UTC()
	s.notes[id] = note
	return note, nil
}

func (s *NoteStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.notes[id]; !ok {
		return ErrNotFound
	}
	delete(s.notes, id)
	return nil
}

func formatID(n uint64) string { return "note-" + strconv.FormatUint(n, 10) }
