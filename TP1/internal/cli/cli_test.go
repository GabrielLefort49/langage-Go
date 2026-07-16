package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"mira/internal/apiclient"
	"mira/internal/cli"
)

func fakeAPI(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/notes", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			var in map[string]string
			_ = json.NewDecoder(r.Body).Decode(&in)
			if in["title"] == "" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]string{"id": "note-1", "title": in["title"], "content": in["content"]}})
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]string{{"id": "note-1", "title": "Go channels", "content": "Channels..."}}})
		}
	})
	mux.HandleFunc("/api/v1/search", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("q") == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]string{{"id": "note-1", "title": "Go channels", "content": "Channels..."}}})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func run(t *testing.T, api *httptest.Server, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	client := apiclient.New(api.URL, 5*time.Second)
	var out, errOut bytes.Buffer
	code = cli.Run(context.Background(), args, client, &out, &errOut)
	return out.String(), errOut.String(), code
}

func TestNoArgsPrintsUsage(t *testing.T) {
	out, _, code := run(t, fakeAPI(t))
	if code != 0 || !strings.Contains(out, "Usage") {
		t.Fatalf("code=%d out=%q", code, out)
	}
}

func TestAdd(t *testing.T) {
	out, _, code := run(t, fakeAPI(t), "add", "Titre", "Contenu")
	if code != 0 {
		t.Fatalf("code=%d", code)
	}
	if !strings.Contains(out, "ajoutée") {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestAddMissingArgs(t *testing.T) {
	out, _, code := run(t, fakeAPI(t), "add", "OnlyTitle")
	if code != 1 || !strings.Contains(out, "Usage") {
		t.Fatalf("code=%d out=%q", code, out)
	}
}

func TestList(t *testing.T) {
	out, _, code := run(t, fakeAPI(t), "list")
	if code != 0 || !strings.Contains(out, "Go channels") {
		t.Fatalf("code=%d out=%q", code, out)
	}
}

func TestSearch(t *testing.T) {
	out, _, code := run(t, fakeAPI(t), "search", "channels")
	if code != 0 || !strings.Contains(out, "Go channels") {
		t.Fatalf("code=%d out=%q", code, out)
	}
}

func TestSearchMissingQuery(t *testing.T) {
	out, _, code := run(t, fakeAPI(t), "search")
	if code != 1 || !strings.Contains(out, "Usage") {
		t.Fatalf("code=%d out=%q", code, out)
	}
}

func TestUnknownCommand(t *testing.T) {
	out, _, code := run(t, fakeAPI(t), "frobnicate")
	if code != 1 || !strings.Contains(out, "Usage") {
		t.Fatalf("code=%d out=%q", code, out)
	}
}
