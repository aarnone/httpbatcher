// +build integration

package test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/aarnone/httpbatcher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestRoundTrip(t *testing.T) {
	resetMock(dockerHostname + ":9001")
	resetMock(dockerHostname + ":9002")

	// given a valid multipart request
	r1, err := http.NewRequest(http.MethodGet, "http://serverA:8080/some/thing", nil)
	require.Nil(t, err)
	r1.Header.Set("Accept", "text/plain")

	r2, err := http.NewRequest(http.MethodPost, "http://serverB:8080/weeee", strings.NewReader("some body"))
	require.Nil(t, err)
	r2.Header.Set("Content-Type", "test/test")

	fatRequest, err := httpbatcher.BuildRequest([]*http.Request{r1, r2}, "http://"+dockerHostname+":8080/batch")
	require.Nil(t, err)

	// and
	setMock(dockerHostname+":9001", `{
	"request" : {
		"method" :  "GET",
		"url" : "/some/thing",
		"headers" : {
			"Accept" : {
			 "equalTo" : "text/plain"
			}
		}
	},
	"response" : {
		"status" : 200,
		"body" : "RESPONSE_BODY",
		"headers" : {
			"Content-Type" : "text/plain",
			"Custom-Header" : "custom-value"
		}
	}
}`)
	setMock(dockerHostname+":9002", `{
	"request" : {
		"method" :  "POST",
		"url" : "/weeee",
		"headers" : {
			"Content-Type" : {
				"equalTo" : "test/test"
			}
		}
	},
	"response" : {
		"status" : 504
	}
}`)

	logHttp(t, fatRequest)

	// when post them to the httpbatcher
	fatResponse, err := http.DefaultClient.Do(fatRequest)
	require.Nil(t, err)

	logHttp(t, fatResponse)

	// then status code is 200
	assert.Equal(t, http.StatusOK, fatResponse.StatusCode)

	// and the response can be unpacked
	responses, err := httpbatcher.UnpackResponse(fatResponse)
	require.Nil(t, err)

	// and status are returned
	assert.Equal(t, 200, responses[0].StatusCode)
	assert.Equal(t, 504, responses[1].StatusCode)

	// and body is returned
	body, _ := ioutil.ReadAll(responses[0].Body)
	assert.Equal(t, "RESPONSE_BODY", string(body))

	// and headers are returned
	assert.Equal(t, "text/plain", responses[0].Header.Get("Content-Type"))
	assert.Equal(t, "custom-value", responses[0].Header.Get("Custom-Header"))

	resetMock(dockerHostname + ":9001")
	resetMock(dockerHostname + ":9002")
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

func setMock(host string, mapdef string) {
	newMappingURL := fmt.Sprintf("http://%v/__admin/mappings/new", host)
	_, err := http.Post(newMappingURL, "application/json", strings.NewReader(mapdef))
	if err != nil {
		panic(err)
	}
}

func resetMock(host string) {
	newMappingURL := fmt.Sprintf("http://%v/__admin/reset", host)
	http.Post(newMappingURL, "", nil)
}
