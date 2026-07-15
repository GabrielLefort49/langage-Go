package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8090"
	}
	addr := ":" + port
	fs := http.FileServer(http.Dir("."))
	http.Handle("/", fs)
	log.Printf("swagger ui server listening on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}
