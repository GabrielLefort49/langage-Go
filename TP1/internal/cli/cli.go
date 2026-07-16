// Package cli implements the mira command-line commands (add, list, search)
// against the mira HTTP API.
package cli

import (
	"context"
	"fmt"
	"io"

	"mira/internal/apiclient"
)

const usage = `Usage :
mira add "titre" "contenu"
mira list
mira search <query>
`

// Run executes the CLI command in args (excluding the program name) and
// returns the process exit code. Output is written to out, errors to errOut.
func Run(ctx context.Context, args []string, client *apiclient.Client, out, errOut io.Writer) int {
	if len(args) < 1 {
		fmt.Fprint(out, usage)
		return 0
	}

	switch args[0] {
	case "add":
		if len(args) < 3 {
			fmt.Fprint(out, usage)
			return 1
		}
		if err := client.AddNote(ctx, args[1], args[2]); err != nil {
			fmt.Fprintln(errOut, "erreur:", err)
			return 1
		}
		fmt.Fprintln(out, "Note ajoutée (via API).")
		return 0

	case "list":
		notes, err := client.ListNotes(ctx)
		if err != nil {
			fmt.Fprintln(errOut, "erreur:", err)
			return 1
		}
		printNotes(out, notes)
		return 0

	case "search":
		if len(args) < 2 {
			fmt.Fprint(out, usage)
			return 1
		}
		notes, err := client.SearchNotes(ctx, args[1])
		if err != nil {
			fmt.Fprintln(errOut, "erreur:", err)
			return 1
		}
		printNotes(out, notes)
		return 0

	default:
		fmt.Fprint(out, usage)
		return 1
	}
}

func printNotes(out io.Writer, notes []apiclient.Note) {
	for _, n := range notes {
		fmt.Fprintf(out, "%s %s\n", n.ID, n.Title)
		fmt.Fprintln(out, n.Content)
		fmt.Fprintln(out)
	}
}
