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

func TestHandlerFailIfContentTypeMalformed(t *testing.T) {
	// given an httpbatcher
	batcherHandler := New()

	// and a request with a maflormed content type
	req, err := http.NewRequest("POST", "localhost", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Content-Type", "")

	// when
	rr := httptest.NewRecorder()
	batcherHandler.ServeHTTP(rr, req)

	// then
	assert.Equal(t, http.StatusUnsupportedMediaType, rr.Code)
	assert.Equal(t, "Content-Type malformed", rr.Body.String())
}

func TestHandlerRejectWrongContentType(t *testing.T) {
	// given an httpbatcher
	batcherHandler := New()

	// and a request with a wrong content type
	req, err := http.NewRequest("POST", "localhost", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Content-Type", "text/plain")

	// when
	rr := httptest.NewRecorder()
	batcherHandler.ServeHTTP(rr, req)

	// then
	assert.Equal(t, http.StatusUnsupportedMediaType, rr.Code)
	assert.Equal(t, "Content-Type must be multipart/mixed", rr.Body.String())
}

func TestHandlerRejectMultipartMixedWithoutBoundary(t *testing.T) {
	// given an httpbatcher
	batcherHandler := New()

	// and a request with a wrong content type
	req, err := http.NewRequest("POST", "localhost", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Content-Type", "multipart/mixed")

	// when
	rr := httptest.NewRecorder()
	batcherHandler.ServeHTTP(rr, req)

	// then
	assert.Equal(t, http.StatusUnsupportedMediaType, rr.Code)
	assert.Equal(t, "Content-Type is missing boundary parameter", rr.Body.String())
}
