package httpbatcher

import "net/http"

type batcher struct {
}

// New instatiate an handler to manage http batch requests
func New() http.Handler {
	return &batcher{}
}

func (b *batcher) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
