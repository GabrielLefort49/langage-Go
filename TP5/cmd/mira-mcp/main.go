// Command mira-mcp is a Model Context Protocol (MCP) server that exposes the
// user's mira notes to AI agents such as Claude Code or Claude Desktop.
//
// It speaks JSON-RPC 2.0 over the stdio transport. Because stdout is reserved
// for the protocol, all diagnostics are written to stderr via slog.
//
// Every tool call is forwarded to the mira HTTP API (never to the database
// directly), which guarantees that the automatic enrichment pipeline runs for
// notes created through this server.
//
// Configuration (environment variables):
//
//	MIRA_API_URL   base URL of the mira API      (default http://localhost:8084)
//	MIRA_TIMEOUT   per-request timeout, Go dur.  (default 15s)
//	MIRA_ENRICH_WAIT  max wait for enrichment     (default 5s)
//	MIRA_LOG_LEVEL debug|info|warn|error         (default info)
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"mira/tp5/internal/mcpserver"
	"mira/tp5/internal/mira"
)

// version is reported to MCP clients; override at build time with
// -ldflags "-X main.version=..." if desired.
var version = "1.0.0"

func main() {
	logger := newLogger()

	apiURL := getenv("MIRA_API_URL", "http://localhost:8084")
	timeout := getenvDuration(logger, "MIRA_TIMEOUT", 15*time.Second)
	enrichWait := getenvDuration(logger, "MIRA_ENRICH_WAIT", 5*time.Second)

	client := mira.New(apiURL, timeout)

	server := mcpserver.New(mcpserver.Config{
		Client:         client,
		Logger:         logger,
		EnrichmentWait: enrichWait,
		Version:        version,
	})

	// Stop cleanly on Ctrl-C / SIGTERM so the stdio transport shuts down.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.Info("mira-mcp starting",
		"version", version,
		"api_url", apiURL,
		"timeout", timeout.String(),
		"enrich_wait", enrichWait.String(),
	)

	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		if ctx.Err() != nil {
			logger.Info("mira-mcp stopped")
			return
		}
		logger.Error("mira-mcp exited with error", "error", err)
		os.Exit(1)
	}
	logger.Info("mira-mcp stopped")
}

// newLogger builds a structured logger writing to stderr. stdout must stay
// clean for the JSON-RPC stdio transport.
func newLogger() *slog.Logger {
	level := slog.LevelInfo
	switch os.Getenv("MIRA_LOG_LEVEL") {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getenvDuration(logger *slog.Logger, key string, fallback time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		logger.Warn("invalid duration, using default", "key", key, "value", raw, "default", fallback.String())
		return fallback
	}
	return d
}
