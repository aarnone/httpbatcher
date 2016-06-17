package httpbatcher

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"golang.org/x/net/context"
)

func TestContextHandlerFunc(t *testing.T) {
	actualArgs := struct {
		Context        context.Context
		ResponseWriter http.ResponseWriter
		Request        *http.Request
	}{}

	chf := contextHandlerFunc(func(c context.Context, w http.ResponseWriter, r *http.Request) {
		actualArgs.Context = c
		actualArgs.ResponseWriter = w
		actualArgs.Request = r
	})

	c := context.WithValue(nil, "aKey", " aVal")
	w := httptest.NewRecorder()
	w.WriteHeader(http.StatusOK)
	r := &http.Request{Host: "somehost"}

	chf.ServeHTTP(c, w, r)

	assert.Equal(t, c, actualArgs.Context)
	assert.Equal(t, w, actualArgs.ResponseWriter)
	assert.Equal(t, r, actualArgs.Request)
}

func TestInitHTTPHandler(t *testing.T) {
	actualArgs := struct {
		Context        context.Context
		ResponseWriter http.ResponseWriter
		Request        *http.Request
	}{}

	chf := contextHandlerFunc(func(c context.Context, w http.ResponseWriter, r *http.Request) {
		actualArgs.Context = c
		actualArgs.ResponseWriter = w
		actualArgs.Request = r
	})

	HTTPHandler := initHTTPHandler(chf)

	w := httptest.NewRecorder()
	w.WriteHeader(http.StatusOK)
	r := &http.Request{Host: "somehost"}

	HTTPHandler.ServeHTTP(w, r)

	assert.Equal(t, w, actualArgs.ResponseWriter)
	assert.Equal(t, r, actualArgs.Request)
	assert.Equal(t, context.TODO(), actualArgs.Context)
}
