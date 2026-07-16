// Package mcpserver wires the mira notes API onto a Model Context Protocol
// server. It declares four tools (search_notes, get_note, add_note and
// list_recent_notes) and delegates all work to the mira HTTP API.
package mcpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"mira/tp5/internal/mira"
)

// Config controls the behaviour of the MCP server.
type Config struct {
	// Client talks to the mira HTTP API.
	Client *mira.Client
	// Logger writes diagnostics to stderr (never stdout, which is reserved for
	// the JSON-RPC stdio transport).
	Logger *slog.Logger
	// EnrichmentWait is the maximum time add_note waits for the asynchronous
	// enrichment pipeline to finish before returning the created note.
	EnrichmentWait time.Duration
	// Version is reported to MCP clients during initialization.
	Version string
}

// New builds an *mcp.Server with all mira tools registered.
func New(cfg Config) *mcp.Server {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.Version == "" {
		cfg.Version = "dev"
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "mira-mcp",
		Title:   "Mira notes",
		Version: cfg.Version,
	}, nil)

	t := &tools{cfg: cfg}
	t.register(server)
	return server
}

type tools struct {
	cfg Config
}

func (t *tools) client() *mira.Client { return t.cfg.Client }
func (t *tools) log() *slog.Logger    { return t.cfg.Logger }

// ---------------------------------------------------------------------------
// Tool input / output types
// ---------------------------------------------------------------------------

type searchInput struct {
	Query string `json:"query" jsonschema:"The natural-language search query. Matched against note titles and content using mira's hybrid full-text plus vector search."`
	Limit int    `json:"limit,omitempty" jsonschema:"Maximum number of notes to return. Defaults to 10 when omitted."`
}

type searchOutput struct {
	Query string      `json:"query"`
	Count int         `json:"count"`
	Notes []mira.Note `json:"notes"`
}

type getInput struct {
	ID string `json:"id" jsonschema:"The unique identifier of the note to retrieve, as returned by search_notes, add_note or list_recent_notes (e.g. note-1737045000000000000)."`
}

type addInput struct {
	Title   string   `json:"title" jsonschema:"Short, descriptive title of the note."`
	Content string   `json:"content" jsonschema:"Full body of the note in plain text or Markdown."`
	Tags    []string `json:"tags,omitempty" jsonschema:"Optional list of tags the user wants attached to the note. They are folded into the note body so they persist and remain searchable; mira also generates its own tags during enrichment."`
}

type addOutput struct {
	Note            mira.Note `json:"note"`
	EnrichmentReady bool      `json:"enrichment_ready"`
	Message         string    `json:"message"`
}

type listInput struct {
	Limit int `json:"limit,omitempty" jsonschema:"Maximum number of notes to return. Defaults to 10 when omitted."`
}

type listOutput struct {
	Count int         `json:"count"`
	Notes []mira.Note `json:"notes"`
}

// ---------------------------------------------------------------------------
// Registration
// ---------------------------------------------------------------------------

func (t *tools) register(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:  "search_notes",
		Title: "Search notes",
		Description: "Search the user's mira notes with a natural-language query. " +
			"Runs mira's hybrid search (full-text ranking combined with vector similarity) " +
			"and returns the best-matching notes ordered by relevance score. " +
			"Use this to find existing notes before answering a question or before creating a duplicate note. " +
			"Returns note ids that can be passed to get_note for the full content.",
		InputSchema: mustSchema[searchInput](func(s *jsonschema.Schema) {
			requireNonEmpty(s, "query")
			withIntDefault(s, "limit", 10, 1, 100)
		}),
	}, t.searchNotes)

	mcp.AddTool(s, &mcp.Tool{
		Name:  "get_note",
		Title: "Get a note",
		Description: "Retrieve a single mira note by its id, including its full content, " +
			"auto-generated tags, short summary and enrichment status. " +
			"Call this after search_notes or list_recent_notes to read a note in full.",
		InputSchema: mustSchema[getInput](func(s *jsonschema.Schema) {
			requireNonEmpty(s, "id")
		}),
	}, t.getNote)

	mcp.AddTool(s, &mcp.Tool{
		Name:  "add_note",
		Title: "Add a note",
		Description: "Create a new note in the user's mira memory. " +
			"The note is stored through the mira API, which automatically enriches it " +
			"(tags, short summary and a search embedding) in the background. " +
			"This tool waits briefly for that enrichment to complete and returns the final note, " +
			"including its enrichment_status. Use it to save facts, summaries or decisions the user asks you to remember.",
		InputSchema: mustSchema[addInput](func(s *jsonschema.Schema) {
			requireNonEmpty(s, "title")
			requireNonEmpty(s, "content")
		}),
	}, t.addNote)

	mcp.AddTool(s, &mcp.Tool{
		Name:  "list_recent_notes",
		Title: "List recent notes",
		Description: "List the most recently created mira notes, newest first. " +
			"Use this to give the user an overview of their latest notes without a specific search query.",
		InputSchema: mustSchema[listInput](func(s *jsonschema.Schema) {
			withIntDefault(s, "limit", 10, 1, 100)
		}),
	}, t.listRecentNotes)
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

func (t *tools) searchNotes(ctx context.Context, _ *mcp.CallToolRequest, in searchInput) (*mcp.CallToolResult, searchOutput, error) {
	query := strings.TrimSpace(in.Query)
	if query == "" {
		return nil, searchOutput{}, errors.New("query must not be empty")
	}
	limit := normalizeLimit(in.Limit)

	notes, err := t.client().Search(ctx, query, limit)
	if err != nil {
		return nil, searchOutput{}, t.wrap("search_notes", err)
	}

	out := searchOutput{Query: query, Count: len(notes), Notes: notes}
	t.log().Info("search_notes", "query", query, "results", len(notes))

	text := fmt.Sprintf("Found %d note(s) matching %q.", len(notes), query)
	for _, n := range notes {
		text += "\n" + noteLine(n)
	}
	return textResult(text), out, nil
}

func (t *tools) getNote(ctx context.Context, _ *mcp.CallToolRequest, in getInput) (*mcp.CallToolResult, mira.Note, error) {
	id := strings.TrimSpace(in.ID)
	if id == "" {
		return nil, mira.Note{}, errors.New("id must not be empty")
	}

	note, err := t.client().Get(ctx, id)
	if err != nil {
		return nil, mira.Note{}, t.wrap("get_note", err)
	}

	t.log().Info("get_note", "id", id, "status", note.EnrichmentStatus)
	return textResult(formatNote(note)), note, nil
}

func (t *tools) addNote(ctx context.Context, _ *mcp.CallToolRequest, in addInput) (*mcp.CallToolResult, addOutput, error) {
	title := strings.TrimSpace(in.Title)
	content := strings.TrimSpace(in.Content)
	if title == "" {
		return nil, addOutput{}, errors.New("title must not be empty")
	}
	if content == "" {
		return nil, addOutput{}, errors.New("content must not be empty")
	}

	// The mira create endpoint accepts only {title, content}. User-supplied tags
	// are appended to the body so they are persisted and remain searchable; mira
	// independently generates its own tags during enrichment.
	body := content
	if tags := cleanTags(in.Tags); len(tags) > 0 {
		body += "\n\nTags: " + strings.Join(tags, ", ")
	}

	note, err := t.client().Create(ctx, mira.CreateNoteInput{Title: title, Content: body})
	if err != nil {
		return nil, addOutput{}, t.wrap("add_note", err)
	}
	t.log().Info("add_note", "id", note.ID, "title", title)

	// Wait briefly for the asynchronous enrichment to finish so we can return the
	// final, enriched note to the agent.
	final, ready := t.waitForEnrichment(ctx, note.ID)
	if ready {
		note = final
	}

	msg := fmt.Sprintf("Created note %s.", note.ID)
	if ready {
		msg += " Enrichment complete."
	} else {
		msg += " Enrichment is still in progress; call get_note later to see the tags and summary."
	}
	out := addOutput{Note: note, EnrichmentReady: ready, Message: msg}
	return textResult(msg + "\n" + formatNote(note)), out, nil
}

func (t *tools) listRecentNotes(ctx context.Context, _ *mcp.CallToolRequest, in listInput) (*mcp.CallToolResult, listOutput, error) {
	limit := normalizeLimit(in.Limit)

	notes, err := t.client().ListRecent(ctx, limit)
	if err != nil {
		return nil, listOutput{}, t.wrap("list_recent_notes", err)
	}

	out := listOutput{Count: len(notes), Notes: notes}
	t.log().Info("list_recent_notes", "results", len(notes))

	text := fmt.Sprintf("%d recent note(s):", len(notes))
	for _, n := range notes {
		text += "\n" + noteLine(n)
	}
	return textResult(text), out, nil
}

// waitForEnrichment polls get_note until the note's enrichment_status is "done",
// or the configured timeout elapses. It returns the latest note observed and
// whether enrichment finished.
func (t *tools) waitForEnrichment(ctx context.Context, id string) (mira.Note, bool) {
	deadline := time.Now().Add(t.cfg.EnrichmentWait)
	var last mira.Note
	for {
		note, err := t.client().Get(ctx, id)
		if err == nil {
			last = note
			if note.EnrichmentStatus == "done" {
				return note, true
			}
		}
		if time.Now().After(deadline) || ctx.Err() != nil {
			return last, false
		}
		select {
		case <-ctx.Done():
			return last, false
		case <-time.After(200 * time.Millisecond):
		}
	}
}

// wrap turns an API/transport error into a clean, agent-facing error. The
// low-level detail is logged to stderr; the returned message stays concise.
func (t *tools) wrap(tool string, err error) error {
	t.log().Error("tool failed", "tool", tool, "error", err)
	var apiErr *mira.APIError
	if errors.As(err, &apiErr) {
		return fmt.Errorf("mira API returned an error: %s", apiErr.Message)
	}
	return fmt.Errorf("could not reach the mira API: %v", err)
}
