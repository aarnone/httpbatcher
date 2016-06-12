package httpbatcher

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

var httpMethods = []string{
	http.MethodGet,
	http.MethodHead,
	http.MethodPost,
	http.MethodPut,
	http.MethodPatch,
	http.MethodDelete,
	http.MethodConnect,
	http.MethodOptions,
	http.MethodTrace,
}

func TestHandlerInterface(t *testing.T) {
	var _ http.Handler = &batcher{}
}

func TestHandlerRejectNonPOSTMethods(t *testing.T) {
	// given an httpbatcher
	batcherHandler := New()

	// when I pass a non POST request
	for _, method := range httpMethods {
		if method != http.MethodPost {
			rr := httptest.NewRecorder()
			req, err := http.NewRequest(method, "localhost", nil)
			if err != nil {
				t.Fatal(err)
			}

			batcherHandler.ServeHTTP(rr, req)

			// then the error code is
			assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
		}
	}
}
