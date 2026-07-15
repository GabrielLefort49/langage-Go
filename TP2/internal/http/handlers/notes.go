package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"mira/tp2/internal/core"
	"mira/tp2/internal/store"
)

type NotesHandler struct{ store *store.NoteStore }

func NewNotesHandler(store *store.NoteStore) *NotesHandler { return &NotesHandler{store: store} }

func (h *NotesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/notes")
	if path == "" || path == "/" {
		switch r.Method {
		case http.MethodPost:
			h.create(w, r)
		case http.MethodGet:
			h.list(w, r)
		default:
			methodNotAllowed(w)
		}
		return
	}
	if !strings.HasPrefix(path, "/") || strings.Contains(strings.TrimPrefix(path, "/"), "/") {
		notFound(w)
		return
	}
	id := strings.TrimPrefix(path, "/")
	switch r.Method {
	case http.MethodGet:
		h.get(w, id)
	case http.MethodPatch:
		h.update(w, r, id)
	case http.MethodDelete:
		h.delete(w, id)
	default:
		methodNotAllowed(w)
	}
}

// create godoc
// @Summary Créer une note
// @Tags notes
// @Accept json
// @Produce json
// @Param note body core.CreateNoteInput true "Nouvelle note"
// @Success 201 {object} NoteResponseDoc
// @Failure 400 {object} ErrorResponseDoc
// @Router /api/v1/notes [post]
func (h *NotesHandler) create(w http.ResponseWriter, r *http.Request) {
	var input core.CreateNoteInput
	if !decodeJSON(w, r, &input) {
		return
	}
	if err := validateCreate(input); err != "" {
		badRequest(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, envelope{Data: h.store.Create(input)})
}

// list godoc
// @Summary Lister les notes
// @Tags notes
// @Produce json
// @Param limit query int false "Nombre de résultats (1 à 100)" default(20)
// @Param offset query int false "Position de départ" default(0)
// @Success 200 {object} NotesResponseDoc
// @Failure 400 {object} ErrorResponseDoc
// @Router /api/v1/notes [get]
func (h *NotesHandler) list(w http.ResponseWriter, r *http.Request) {
	limit, offset, ok := pagination(w, r)
	if !ok {
		return
	}
	notes, total := h.store.List(limit, offset)
	writeJSON(w, http.StatusOK, envelope{Data: notes, Meta: &meta{Total: total, Limit: limit, Offset: offset}})
}

// get godoc
// @Summary Récupérer une note
// @Tags notes
// @Produce json
// @Param id path string true "Identifiant de note"
// @Success 200 {object} NoteResponseDoc
// @Failure 404 {object} ErrorResponseDoc
// @Router /api/v1/notes/{id} [get]
func (h *NotesHandler) get(w http.ResponseWriter, id string) {
	note, err := h.store.Get(id)
	if errors.Is(err, store.ErrNotFound) {
		notFound(w)
		return
	}
	writeJSON(w, http.StatusOK, envelope{Data: note})
}

// update godoc
// @Summary Modifier partiellement une note
// @Tags notes
// @Accept json
// @Produce json
// @Param id path string true "Identifiant de note"
// @Param note body core.UpdateNoteInput true "Champs à modifier"
// @Success 200 {object} NoteResponseDoc
// @Failure 400 {object} ErrorResponseDoc
// @Failure 404 {object} ErrorResponseDoc
// @Router /api/v1/notes/{id} [patch]
func (h *NotesHandler) update(w http.ResponseWriter, r *http.Request, id string) {
	var input core.UpdateNoteInput
	if !decodeJSON(w, r, &input) {
		return
	}
	if input.Title == nil && input.Content == nil {
		badRequest(w, "at least one field must be supplied")
		return
	}
	if input.Title != nil && strings.TrimSpace(*input.Title) == "" {
		badRequest(w, "title must not be empty")
		return
	}
	if input.Title != nil && len(*input.Title) > 200 {
		badRequest(w, "title must be at most 200 characters")
		return
	}
	if input.Content != nil && len(*input.Content) > 10000 {
		badRequest(w, "content must be at most 10000 characters")
		return
	}
	note, err := h.store.Update(id, input)
	if errors.Is(err, store.ErrNotFound) {
		notFound(w)
		return
	}
	writeJSON(w, http.StatusOK, envelope{Data: note})
}

// delete godoc
// @Summary Supprimer une note
// @Tags notes
// @Param id path string true "Identifiant de note"
// @Success 204
// @Failure 404 {object} ErrorResponseDoc
// @Router /api/v1/notes/{id} [delete]
func (h *NotesHandler) delete(w http.ResponseWriter, id string) {
	if err := h.store.Delete(id); errors.Is(err, store.ErrNotFound) {
		notFound(w)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Search godoc
// @Summary Rechercher des notes
// @Tags notes
// @Produce json
// @Param q query string true "Texte recherché"
// @Param limit query int false "Nombre de résultats (1 à 100)" default(20)
// @Param offset query int false "Position de départ" default(0)
// @Success 200 {object} NotesResponseDoc
// @Failure 400 {object} ErrorResponseDoc
// @Router /api/v1/search [get]
func (h *NotesHandler) Search(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		badRequest(w, "q is required")
		return
	}
	limit, offset, ok := pagination(w, r)
	if !ok {
		return
	}
	notes, total := h.store.Search(query, limit, offset)
	writeJSON(w, http.StatusOK, envelope{Data: notes, Meta: &meta{Total: total, Limit: limit, Offset: offset}})
}

type envelope struct {
	Data  any       `json:"data,omitempty"`
	Error *apiError `json:"error,omitempty"`
	Meta  *meta     `json:"meta,omitempty"`
}
type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
type meta struct {
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

func writeJSON(w http.ResponseWriter, status int, body envelope) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, envelope{Error: &apiError{Code: code, Message: message}})
}
func badRequest(w http.ResponseWriter, message string) {
	writeError(w, http.StatusBadRequest, "bad_request", message)
}
func notFound(w http.ResponseWriter) {
	writeError(w, http.StatusNotFound, "not_found", "note not found")
}
func methodNotAllowed(w http.ResponseWriter) {
	writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	defer r.Body.Close()
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		badRequest(w, "invalid JSON payload")
		return false
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		badRequest(w, "request body must contain one JSON object")
		return false
	}
	return true
}
func validateCreate(input core.CreateNoteInput) string {
	if strings.TrimSpace(input.Title) == "" {
		return "title is required"
	}
	if len(input.Title) > 200 {
		return "title must be at most 200 characters"
	}
	if len(input.Content) > 10000 {
		return "content must be at most 10000 characters"
	}
	return ""
}
func pagination(w http.ResponseWriter, r *http.Request) (int, int, bool) {
	limit, offset := 20, 0
	var err error
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > 100 {
			badRequest(w, "limit must be between 1 and 100")
			return 0, 0, false
		}
	}
	if raw := r.URL.Query().Get("offset"); raw != "" {
		offset, err = strconv.Atoi(raw)
		if err != nil || offset < 0 {
			badRequest(w, "offset must be a non-negative integer")
			return 0, 0, false
		}
	}
	return limit, offset, true
}
