// Package apiclient is a small HTTP client for the mira notes API. The CLI
// always goes through this client rather than touching storage directly, so
// that the API's automatic enrichment pipeline is triggered for every note.
package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Note is the subset of the mira note representation the CLI displays.
type Note struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

type listEnvelope struct {
	Data []Note `json:"data"`
}

// Client talks to the mira HTTP API.
type Client struct {
	baseURL string
	http    *http.Client
}

// New builds a client for the given base URL (e.g. http://localhost:8080).
func New(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: timeout},
	}
}

// AddNote creates a new note through the API.
func (c *Client) AddNote(ctx context.Context, title, content string) error {
	payload, err := json.Marshal(map[string]string{"title": title, "content": content})
	if err != nil {
		return fmt.Errorf("encode payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/notes", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("call mira API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return apiError(resp)
	}
	return nil
}

// ListNotes returns the most recent notes.
func (c *Client) ListNotes(ctx context.Context) ([]Note, error) {
	return c.getNotes(ctx, "/api/v1/notes?limit=100&offset=0")
}

// SearchNotes returns notes matching the given query.
func (c *Client) SearchNotes(ctx context.Context, query string) ([]Note, error) {
	q := url.Values{"q": {query}, "limit": {"20"}, "offset": {"0"}}
	return c.getNotes(ctx, "/api/v1/search?"+q.Encode())
}

func (c *Client) getNotes(ctx context.Context, path string) ([]Note, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call mira API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, apiError(resp)
	}

	var env listEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return env.Data, nil
}

func apiError(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return fmt.Errorf("mira API returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
}
