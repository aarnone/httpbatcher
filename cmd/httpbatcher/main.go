package main

import (
	"fmt"
	"log"
	"mime"
	"net/http"

	"github.com/aarnone/httpbatcher/split"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/batch", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil {
			log.Printf("Error parsing the media type: %s", err)
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Wrong media type")
		}

		if mediaType != "multipart/mixed" {
			w.WriteHeader(http.StatusUnsupportedMediaType)
			return
		}

		requests, err := split.UnpackRequests(r)
		if err != nil {
			log.Printf("Error unpacking the request: %s", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		responses := make([]*http.Response, 0, len(requests))
		for _, req := range requests {
			resp, rErr := http.DefaultClient.Do(req)
			if rErr != nil {
				log.Printf("Error during the fanout requests: %s", rErr)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			responses = append(responses, resp)
		}

		if err := split.WriteResponses(w, responses); err != nil {
			log.Printf("Error packing the responses: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	})

	log.Fatal(http.ListenAndServe(":8080", mux))
}
