package httpbatcher

import (
	"net/http"

	"golang.org/x/net/context"
)

type contextHandler interface {
	ServeHTTP(c context.Context, w http.ResponseWriter, r *http.Request)
}

type contextHandlerFunc func(c context.Context, w http.ResponseWriter, r *http.Request)

func (f contextHandlerFunc) ServeHTTP(c context.Context, w http.ResponseWriter, r *http.Request) {
	f(c, w, r)
}

func initHTTPHandler(ch contextHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ch.ServeHTTP(context.TODO(), w, r)
	})
}
