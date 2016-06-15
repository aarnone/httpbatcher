package httpbatcher

import (
	"mime"
	"net/http"
)

type batcher struct {
}

// New instatiate an handler to manage http batch requests
func New() http.Handler {
	return &batcher{}
}

func validateBatchRequestHandler(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		if params["boundary"] == "" {
			w.WriteHeader(http.StatusUnsupportedMediaType)
			w.Write([]byte("Content-Type is missing boundary parameter"))
			return
		}

		next(w, r)
	}
}

func (b *batcher) ServeHTTP(w http.ResponseWriter, r *http.Request) {
}
