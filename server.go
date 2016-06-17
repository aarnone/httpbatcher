package httpbatcher

import (
	"io"
	"log"
	"mime"
	"net/http"

	"golang.org/x/net/context"
)

type batcher struct {
}

type requestExtractor interface {
	Extract(boundary string, body io.Reader) ([]*http.Request, error)
}

// New instatiate an handler to manage http batch requests
func New() http.Handler {
	return &batcher{}
}

func validateBatchRequestHandler(next contextHandlerFunc) contextHandlerFunc {
	return func(c context.Context, w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}

		mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil {
			w.WriteHeader(http.StatusUnsupportedMediaType)
			w.Write([]byte("Content-Type malformed"))
			return
		}
		if mediaType != "multipart/mixed" {
			w.WriteHeader(http.StatusUnsupportedMediaType)
			w.Write([]byte("Content-Type must be multipart/mixed"))
			return
		}

		boundary := params["boundary"]
		if boundary == "" {
			w.WriteHeader(http.StatusUnsupportedMediaType)
			w.Write([]byte("Content-Type is missing boundary parameter"))
			return
		}

		next(context.WithValue(c, "boundary", boundary), w, r)
	}
}

func extractPayloadHandler(extractor requestExtractor, next contextHandlerFunc) contextHandlerFunc {
	return func(c context.Context, w http.ResponseWriter, r *http.Request) {
		requests, err := extractor.Extract(c.Value("boundary").(string), r.Body)
		if err != nil {
			log.Println("Requests extraction failed")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		next(context.WithValue(c, "payload", requests), w, r)
	}
}

func (b *batcher) ServeHTTP(w http.ResponseWriter, r *http.Request) {
}
