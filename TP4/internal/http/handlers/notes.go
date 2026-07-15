package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"mira/tp4/internal/core"
	"mira/tp4/internal/enricher"
	pgstore "mira/tp4/internal/store/postgres"
)

type NotesHandler struct {
	store *pgstore.Store
	enr   *enricher.Enricher
}

func NewNotesHandler(store *pgstore.Store, enr *enricher.Enricher) *NotesHandler {
	return &NotesHandler{store: store, enr: enr}
}

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

func (h *NotesHandler) create(w http.ResponseWriter, r *http.Request) {
	var input core.CreateNoteInput
	if !decodeJSON(w, r, &input) {
		return
	}
	if strings.TrimSpace(input.Title) == "" {
		badRequest(w, "title is required")
		return
	}
	note, err := h.store.Create(context.Background(), input)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	// publish job
	h.enr.Publish(note.ID)
	writeJSON(w, http.StatusCreated, envelope{Data: note})
}

func (h *NotesHandler) list(w http.ResponseWriter, r *http.Request) {
	limit, offset, ok := pagination(w, r)
	if !ok {
		return
	}
	notes, total, err := h.store.List(context.Background(), limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{Data: notes, Meta: &meta{Total: total, Limit: limit, Offset: offset}})
}

func (h *NotesHandler) get(w http.ResponseWriter, id string) {
	note, err := h.store.Get(context.Background(), id)
	if err != nil {
		notFound(w)
		return
	}
	writeJSON(w, http.StatusOK, envelope{Data: note})
}

func (h *NotesHandler) update(w http.ResponseWriter, r *http.Request, id string) {
	var input core.UpdateNoteInput
	if !decodeJSON(w, r, &input) {
		return
	}
	if input.Title == nil && input.Content == nil {
		badRequest(w, "at least one field must be supplied")
		return
	}
	note, err := h.store.Update(context.Background(), id, input)
	if err != nil {
		notFound(w)
		return
	}
	// publish enrichment job
	h.enr.Publish(note.ID)
	writeJSON(w, http.StatusOK, envelope{Data: note})
}

func (h *NotesHandler) delete(w http.ResponseWriter, id string) {
	if err := h.store.Delete(context.Background(), id); err != nil {
		notFound(w)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type SearchHandler struct {
	store *pgstore.Store
}

func NewSearchHandler(store *pgstore.Store) *SearchHandler {
	return &SearchHandler{store: store}
}

func (h *SearchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
	notes, total, err := h.store.Search(context.Background(), query, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{Data: notes, Meta: &meta{Total: total, Limit: limit, Offset: offset}})
}

func SwaggerUIHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>mira TP4 Swagger</title>
    <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5.17.2/swagger-ui.css" />
    <style>body{margin:0;background:#f5f7fb;}#swagger-ui{max-width:1200px;margin:0 auto;padding:20px;}</style>
  </head>
  <body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5.17.2/swagger-ui-bundle.js"></script>
    <script>
      window.onload = () => {
        SwaggerUIBundle({
          url: '/swagger.json',
          dom_id: '#swagger-ui',
          deepLinking: true,
          presets: [SwaggerUIBundle.presets.apis]
        });
      };
    </script>
  </body>
</html>`))
}

func SwaggerSpecHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write([]byte(`{
  "openapi": "3.0.0",
  "info": {
    "title": "mira TP4 API",
    "version": "1.0.0"
  },
  "paths": {
    "/api/v1/notes": {
      "post": {"summary": "Create a note", "responses": {"201": {"description": "Created"}}},
      "get": {"summary": "List notes", "responses": {"200": {"description": "OK"}}}
    },
    "/api/v1/notes/{id}": {
      "get": {"summary": "Get a note", "responses": {"200": {"description": "OK"}}},
      "patch": {"summary": "Update a note", "responses": {"200": {"description": "OK"}}},
      "delete": {"summary": "Delete a note", "responses": {"204": {"description": "Deleted"}}}
    },
    "/api/v1/search": {
      "get": {"summary": "Search notes", "parameters": [{"name": "q", "in": "query", "required": true, "schema": {"type": "string"}}], "responses": {"200": {"description": "OK"}}}
    }
  }
}`))
}

// reuse helpers from TP2 style
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
