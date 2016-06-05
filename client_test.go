package httpbatcher

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildRequestWithoutError(t *testing.T) {
	// when
	fatRequest, err := BuildRequest(someRequests(3), "http://batcher/batch")

	// then no error is returned
	assert.Nil(t, err)
	assert.NotNil(t, fatRequest)
}

func TestBuildRequestWithInvalidURL(t *testing.T) {
	// when
	fatRequest, err := BuildRequest(someRequests(3), ":localhost")

	// then no error is returned
	assert.NotNil(t, err)
	assert.Empty(t, fatRequest)
}

func TestBuildRequestCreatesTheWrapperRequest(t *testing.T) {
	// when
	fatRequest, _ := BuildRequest(someRequests(3), "http://batcher/batch")

	// then request is POST http://batcher/batch
	assert.Equal(t, http.MethodPost, fatRequest.Method)
	assert.Equal(t, "http://batcher/batch", fatRequest.URL.String())

	// and the Content-Type is multipart/mixed with boundary set
	mediaType, params, err := mime.ParseMediaType(fatRequest.Header.Get("Content-Type"))
	assert.Nil(t, err)
	assert.Equal(t, "multipart/mixed", mediaType)
	assert.NotEmpty(t, params["boundary"])
}

func TestBuildRequestAllThePartsAreCreated(t *testing.T) {
	// given three requests
	reqs := someRequests(3)

	// when
	fatRequest, _ := BuildRequest(reqs, "")

	// then all the parts are bundled in order
	r := parseMultipartBody(fatRequest)
	for i := 0; i < 3; i++ {
		p, err := r.NextPart()
		assert.Nil(t, err)

		req, err := http.ReadRequest(bufio.NewReader(p))
		assert.Nil(t, err)

		assert.Equal(t, http.MethodGet, req.Method)
		assert.Equal(t, fakeURLFor(i), req.URL.String())
	}
}

func TestBuildRequestWithBody(t *testing.T) {
	// given a request with body
	reqWithBody, _ := http.NewRequest(http.MethodPost, "http://somehost/resource/path", strings.NewReader("some content"))

	// when
	fatRequest, _ := BuildRequest([]*http.Request{reqWithBody}, "localhost")

	// then
	mpr := parseMultipartBody(fatRequest)
	part, _ := mpr.NextPart()

	req, _ := http.ReadRequest(bufio.NewReader(part))

	body, _ := ioutil.ReadAll(req.Body)
	assert.Equal(t, "some content", string(body))
}

func TestBuildRequestWithHeader(t *testing.T) {
	// given a request with body
	reqWithHeader, _ := http.NewRequest(http.MethodPost, "http://somehost/resource/path", nil)
	reqWithHeader.Header.Set("Custom-Header", "custom value")

	// when
	fatRequest, _ := BuildRequest([]*http.Request{reqWithHeader}, "localhost")

	// then
	mpr := parseMultipartBody(fatRequest)
	part, _ := mpr.NextPart()

	req, _ := http.ReadRequest(bufio.NewReader(part))
	value := req.Header.Get("Custom-Header")

	assert.Equal(t, "custom value", value)
}

func TestBuildRequestPreseveSchema(t *testing.T) {
	// given a request with body
	reqHTTPS, _ := http.NewRequest(http.MethodPost, "https://somehost/resource/path", nil)

	// when
	fatRequest, _ := BuildRequest([]*http.Request{reqHTTPS}, "localhost")

	// then
	mpr := parseMultipartBody(fatRequest)
	part, _ := mpr.NextPart()

	req, _ := http.ReadRequest(bufio.NewReader(part))

	assert.Equal(t, "https", req.URL.Scheme)
}

func parseMultipartBody(fatRequest *http.Request) *multipart.Reader {
	_, params, _ := mime.ParseMediaType(fatRequest.Header.Get("Content-Type"))
	boundary := params["boundary"]

	return multipart.NewReader(fatRequest.Body, boundary)
}

func fakeURLFor(i int) string {
	return fmt.Sprintf("http://somehost:%d/resource/path", i)
}

func someRequests(n int) []*http.Request {
	requests := []*http.Request{}
	for i := 0; i < n; i++ {
		url := fakeURLFor(i)
		r, _ := http.NewRequest(http.MethodGet, url, nil)
		requests = append(requests, r)
	}

	return requests
}
