package httpbatcher

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteSerially(t *testing.T) {
	var arg1 = struct {
		method string
		body   string
	}{}
	var arg2 = arg1

	ts1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		arg1.method = r.Method
		body, _ := ioutil.ReadAll(r.Body)
		arg1.body = string(body)
		w.WriteHeader(201)
		fmt.Fprintf(w, "ts1")
	}))
	defer ts1.Close()

	ts2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		arg2.method = r.Method
		body, _ := ioutil.ReadAll(r.Body)
		arg2.body = string(body)
		w.WriteHeader(202)
		fmt.Fprintf(w, "ts2")
	}))
	defer ts2.Close()

	r1, _ := http.NewRequest("PATCH", ts1.URL, strings.NewReader("patch body"))
	r2, _ := http.NewRequest("POST", ts2.URL, strings.NewReader("post body"))

	responses, err := ExecuteSerially([]*http.Request{r1, r2})
	require.Nil(t, err)

	assert.Equal(t, "PATCH", arg1.method)
	assert.Equal(t, "POST", arg2.method)

	assert.Equal(t, "patch body", arg1.body)
	assert.Equal(t, "post body", arg2.body)

	assert.Equal(t, 201, responses[0].StatusCode)
	assert.Equal(t, 202, responses[1].StatusCode)

	body1, _ := ioutil.ReadAll(responses[0].Body)
	body2, _ := ioutil.ReadAll(responses[1].Body)
	assert.Equal(t, "ts1", string(body1))
	assert.Equal(t, "ts2", string(body2))
}

func TestExecuteSeriallyOnError(t *testing.T) {
	r1, _ := http.NewRequest("PATCH", "localhost", strings.NewReader("patch body"))
	r1.URL = nil

	_, err := ExecuteSerially([]*http.Request{r1})
	assert.EqualError(t, err, "error occurred while executing http request: http: nil Request.URL")
}
