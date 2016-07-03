package main

import (
	"log"
	"net/http"

	"github.com/aarnone/httpbatcher"
)

func main() {
	mux := http.NewServeMux()

	mux.Handle("/batch", httpbatcher.New())

	log.Fatal(http.ListenAndServe(":8080", mux))
}
