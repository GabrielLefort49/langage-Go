// Command mira is the CLI client for the mira notes API. It never touches
// storage directly: every command goes through the HTTP API (MIRA_API) so
// that server-side enrichment is triggered.
package main

import (
	"context"
	"os"
	"time"

	"mira/internal/apiclient"
	"mira/internal/cli"
)

func main() {
	api := os.Getenv("MIRA_API")
	if api == "" {
		api = "http://localhost:8080"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client := apiclient.New(api, 15*time.Second)
	os.Exit(cli.Run(ctx, os.Args[1:], client, os.Stdout, os.Stderr))
}
