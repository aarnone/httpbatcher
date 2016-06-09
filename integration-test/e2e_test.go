// +build integration

package test

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/aarnone/httpbatcher"
	"github.com/stretchr/testify/assert"
)

var dockerHostname string

func init() {
	if dockerHost := os.Getenv("DOCKER_HOST"); dockerHost == "" {
		dockerHostname = "localhost"
	} else {
		dockerHostURL, err := url.Parse(dockerHost)
		if err != nil {
			panic(err)
		}

		dockerHostname = strings.Split(dockerHostURL.Host, ":")[0]
	}
}

func TestMixedCalls(t *testing.T) {
	r1, err := http.NewRequest(http.MethodGet, "http://serverA:8080/some/thing", nil)
	if err != nil {
		panic(err)
	}

	r2, err := http.NewRequest(http.MethodPost, "http://serverB:8080/weeee", strings.NewReader("some body"))
	if err != nil {
		panic(err)
	}

	fatRequest, err := httpbatcher.BuildRequest([]*http.Request{r1, r2}, "http://"+dockerHostname+":8080/batch")
	if err != nil {
		panic(err)
	}

	logHttp(t, fatRequest)

	fatResponse, err := http.DefaultClient.Do(fatRequest)
	if err != nil {
		panic(err)
	}

	assert.Equal(t, http.StatusOK, fatResponse.StatusCode)

	logHttp(t, fatResponse)
}

func logHttp(t *testing.T, r interface{}) {
	var dump []byte
	var err error
	switch rt := r.(type) {
	case *http.Response:
		dump, err = httputil.DumpResponse(rt, true)
	case *http.Request:
		dump, err = httputil.DumpRequestOut(rt, true)
	}

	if err != nil {
		t.Log(err)
		return
	}

	t.Log(string(dump))
}
