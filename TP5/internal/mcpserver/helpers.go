package mcpserver

import (
	"fmt"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"mira/tp5/internal/mira"
)

const (
	defaultLimit = 10
	maxLimit     = 100
)

// mustSchema infers a strict JSON schema for T and applies optional tweaks.
// It panics on failure, which is acceptable at server construction time.
func mustSchema[T any](tweaks ...func(*jsonschema.Schema)) *jsonschema.Schema {
	schema, err := jsonschema.For[T](nil)
	if err != nil {
		panic(fmt.Sprintf("mcpserver: infer schema for %T: %v", *new(T), err))
	}
	for _, tw := range tweaks {
		tw(schema)
	}
	return schema
}

// requireNonEmpty marks a string property as required with a minimum length of
// one, so blank values are rejected before reaching the handler.
func requireNonEmpty(s *jsonschema.Schema, name string) {
	if p := s.Properties[name]; p != nil {
		p.MinLength = jsonschema.Ptr(1)
	}
	if !contains(s.Required, name) {
		s.Required = append(s.Required, name)
	}
}

// withIntDefault gives an integer property a default value and inclusive
// bounds, advertising the contract in the schema itself.
func withIntDefault(s *jsonschema.Schema, name string, def, min, max int) {
	p := s.Properties[name]
	if p == nil {
		return
	}
	p.Default = mustRawInt(def)
	p.Minimum = jsonschema.Ptr(float64(min))
	p.Maximum = jsonschema.Ptr(float64(max))
}

func mustRawInt(v int) []byte {
	return []byte(fmt.Sprintf("%d", v))
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// normalizeLimit clamps a requested limit into [1, maxLimit], defaulting when
// the caller omits it (zero) or sends an out-of-range value.
func normalizeLimit(limit int) int {
	if limit <= 0 {
		return defaultLimit
	}
	if limit > maxLimit {
		return maxLimit
	}
	return limit
}

// cleanTags trims and de-duplicates tags, dropping blanks.
func cleanTags(tags []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	return out
}

// textResult builds a CallToolResult carrying a single text block.
func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}
}

// noteLine renders a one-line summary of a note for the human-readable output.
func noteLine(n mira.Note) string {
	line := fmt.Sprintf("- [%s] %s", n.ID, n.Title)
	if n.Score != nil {
		line += fmt.Sprintf(" (score %.3f)", *n.Score)
	}
	return line
}

// formatNote renders a full note as readable text.
func formatNote(n mira.Note) string {
	var b strings.Builder
	fmt.Fprintf(&b, "id: %s\n", n.ID)
	fmt.Fprintf(&b, "title: %s\n", n.Title)
	fmt.Fprintf(&b, "enrichment_status: %s\n", n.EnrichmentStatus)
	if n.ShortSummary != nil && strings.TrimSpace(*n.ShortSummary) != "" {
		fmt.Fprintf(&b, "summary: %s\n", *n.ShortSummary)
	}
	if len(n.Tags) > 0 {
		fmt.Fprintf(&b, "tags: %s\n", strings.Join(n.Tags, ", "))
	}
	fmt.Fprintf(&b, "content:\n%s", n.Content)
	return b.String()
}
