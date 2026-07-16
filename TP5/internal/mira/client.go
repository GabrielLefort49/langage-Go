// Package mira is a thin HTTP client for the mira notes API (TP4).
//
// The MCP server always talks to mira through this HTTP client, never to the
// database directly. Going through the API is what guarantees that the
// asynchronous enrichment pipeline (tags, summary, embedding) is triggered for
// every note that is created or updated.
package mira

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Note mirrors the note representation returned by the mira API.
type Note struct {
	ID               string   `json:"id"`
	Title            string   `json:"title"`
	Content          string   `json:"content"`
	CreatedAt        string   `json:"created_at"`
	UpdatedAt        string   `json:"updated_at"`
	EnrichmentStatus string   `json:"enrichment_status"`
	ShortSummary     *string  `json:"short_summary,omitempty"`
	Score            *float64 `json:"score,omitempty"`
	Tags             []string `json:"tags,omitempty"`
}

// envelope is the response wrapper used by the mira API.
type envelope struct {
	Data  json.RawMessage `json:"data"`
	Error *apiError       `json:"error"`
	Meta  *meta           `json:"meta"`
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

// Client is a small wrapper around net/http targeting the mira API.
type Client struct {
	baseURL string
	http    *http.Client
}

// New builds a client for the given base URL (e.g. http://localhost:8084).
// The provided timeout bounds every individual HTTP request.
func New(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: timeout},
	}
}

// APIError describes a structured error returned by the mira API.
type APIError struct {
	Status  int
	Code    string
	Message string
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("mira API error %d (%s): %s", e.Status, e.Code, e.Message)
	}
	return fmt.Sprintf("mira API error %d: %s", e.Status, e.Message)
}

// Search runs a hybrid (full-text + vector) search and returns matching notes.
func (c *Client) Search(ctx context.Context, query string, limit int) ([]Note, error) {
	q := url.Values{}
	q.Set("q", query)
	q.Set("limit", strconv.Itoa(limit))
	q.Set("offset", "0")

	var notes []Note
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/search?"+q.Encode(), nil, &notes); err != nil {
		return nil, err
	}
	return notes, nil
}

// Get fetches a single note by ID, including its tags, summary and status.
func (c *Client) Get(ctx context.Context, id string) (Note, error) {
	var note Note
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/notes/"+url.PathEscape(id), nil, &note); err != nil {
		return Note{}, err
	}
	return note, nil
}

// CreateNoteInput is the payload accepted by the mira create-note endpoint.
type CreateNoteInput struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// Create creates a new note. The mira API automatically schedules enrichment
// for the returned note.
func (c *Client) Create(ctx context.Context, in CreateNoteInput) (Note, error) {
	var note Note
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/notes", in, &note); err != nil {
		return Note{}, err
	}
	return note, nil
}

// ListRecent returns the most recently created notes (newest first).
func (c *Client) ListRecent(ctx context.Context, limit int) ([]Note, error) {
	q := url.Values{}
	q.Set("limit", strconv.Itoa(limit))
	q.Set("offset", "0")

	var notes []Note
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/notes?"+q.Encode(), nil, &notes); err != nil {
		return nil, err
	}
	return notes, nil
}

// doJSON performs an HTTP request, decodes the mira envelope and unmarshals the
// data payload into out (which may be nil to ignore the body).
func (c *Client) doJSON(ctx context.Context, method, path string, body any, out any) error {
	var reader *bytes.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode request body: %w", err)
		}
		reader = bytes.NewReader(payload)
	} else {
		reader = bytes.NewReader(nil)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("call mira API (%s %s): %w", method, path, err)
	}
	defer resp.Body.Close()

	var env envelope
	// A 204 (no content) carries no body; decoding is skipped for it.
	if resp.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
			return fmt.Errorf("decode mira response (status %d): %w", resp.StatusCode, err)
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		apiErr := &APIError{Status: resp.StatusCode, Message: http.StatusText(resp.StatusCode)}
		if env.Error != nil {
			apiErr.Code = env.Error.Code
			apiErr.Message = env.Error.Message
		}
		return apiErr
	}

	if out != nil && len(env.Data) > 0 {
		if err := json.Unmarshal(env.Data, out); err != nil {
			return fmt.Errorf("decode data payload: %w", err)
		}
	}
	return nil
}
