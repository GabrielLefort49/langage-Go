package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

func usage() {
	fmt.Println("Usage :")
	fmt.Println(`mira add "titre" "contenu"`)
	fmt.Println("mira list")
	fmt.Println("mira search <query>")
}

func main() {

	if len(os.Args) < 2 {
		usage()
		return
	}

	api := os.Getenv("MIRA_API")
	if api == "" {
		api = "http://localhost:8080"
	}

	switch os.Args[1] {

	case "add":
		if len(os.Args) < 4 {
			usage()
			return
		}
		payload := map[string]string{"title": os.Args[2], "content": os.Args[3]}
		b, _ := json.Marshal(payload)
		resp, err := http.Post(api+"/api/v1/notes", "application/json", bytes.NewReader(b))
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			panic(string(body))
		}
		fmt.Println("Note ajoutée (via API).")

	case "list":
		resp, err := http.Get(api + "/api/v1/notes?limit=100&offset=0")
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()
		var out struct {
			Data []struct {
				ID      string `json:"id"`
				Title   string `json:"title"`
				Content string `json:"content"`
			} `json:"data"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&out)
		for _, n := range out.Data {
			fmt.Printf("%s %s\n", n.ID, n.Title)
			fmt.Println(n.Content)
			fmt.Println()
		}

	case "search":
		if len(os.Args) < 3 {
			usage()
			return
		}
		resp, err := http.Get(api + "/api/v1/search?q=" + os.Args[2] + "&limit=20&offset=0")
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()
		var out struct {
			Data []struct {
				ID      string `json:"id"`
				Title   string `json:"title"`
				Content string `json:"content"`
			} `json:"data"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&out)
		for _, n := range out.Data {
			fmt.Printf("%s %s\n", n.ID, n.Title)
			fmt.Println(n.Content)
			fmt.Println()
		}

	default:
		usage()
	}
}
