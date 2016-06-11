package httpbatcher

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildRequestWithoutError(t *testing.T) {
	// when
	fatRequest, err := BuildRequest("http://batcher/batch", someRequests(3)...)

	// then no error is returned
	assert.Nil(t, err)
	assert.NotNil(t, fatRequest)
}

func TestBuildRequestWithInvalidURL(t *testing.T) {
	// when
	fatRequest, err := BuildRequest(":localhost", someRequests(3)...)

	// then no error is returned
	assert.NotNil(t, err)
	assert.Empty(t, fatRequest)
}

func TestBuildRequestCreatesTheWrapperRequest(t *testing.T) {
	// when
	fatRequest, _ := BuildRequest("http://batcher/batch", someRequests(3)...)

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
	fatRequest, _ := BuildRequest("", reqs...)

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
	fatRequest, _ := BuildRequest("localhost", reqWithBody)

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
	fatRequest, _ := BuildRequest("localhost", reqWithHeader)

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
	fatRequest, _ := BuildRequest("localhost", reqHTTPS)

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

func TestUnpackResponseFailContentTypeCannotBeParsed(t *testing.T) {
	// given a response with wrong content type
	fatResponse := aFatResponse(aResponse(""))
	fatResponse.Header.Set("Content-Type", "")

	// when
	responses, err := UnpackResponse(fatResponse)

	// then error is not nil
	assert.EqualError(t, err, "Can't parse the content type: mime: no media type")
	assert.Empty(t, responses)
}

func TestUnpackResponseFailIfNotTheRightContentType(t *testing.T) {
	// given a response with wrong content type
	fatResponse := aFatResponse(aResponse(""))
	fatResponse.Header.Set("Content-Type", "text/plain")

	// when
	responses, err := UnpackResponse(fatResponse)

	// then error is not nil
	assert.EqualError(t, err, ErrInvalidMediaType.Error())
	assert.Empty(t, responses)
}

func TestUnpackResponseFailIfContentTypeDoNotIncludeABoundary(t *testing.T) {
	// given a response with no boundary
	fatResponse := aFatResponse(aResponse(""))
	fatResponse.Header.Set("Content-Type", "multipart/mixed")

	// when
	responses, err := UnpackResponse(fatResponse)

	// then error is not nil
	assert.EqualError(t, err, ErrMultipartBoundaryNotDefined.Error())
	assert.Empty(t, responses)
}

func TestUnpackResponseFailIfStatusNotOk(t *testing.T) {
	// given a failing response
	fatResponse := aResponse("really bad error")
	fatResponse.Status = "400 Bad Request"
	fatResponse.StatusCode = 400

	// when
	responses, err := UnpackResponse(fatResponse)

	// then error report the http message
	assert.EqualError(t, err, "Response status is 400: really bad error")
	assert.Empty(t, responses)
}

func TestUnpackResponseShouldNotFailOnAValidResponse(t *testing.T) {
	// given a valid response
	fatResponse := aFatResponse(aResponse("valid response"))

	// when
	responses, err := UnpackResponse(fatResponse)

	// then
	assert.Nil(t, err)
	assert.NotEmpty(t, responses)
}

func TestUnpackResponseShouldReturnTheResponsesInOrder(t *testing.T) {
	// given a valid response with two parts
	first, second := aResponse("first"), aResponse("second")
	fatResponse := aFatResponse(first, second)

	// when
	responses, _ := UnpackResponse(fatResponse)

	// then
	firstBody, _ := ioutil.ReadAll(responses[0].Body)
	assert.Equal(t, "first", string(firstBody))

	secondBody, _ := ioutil.ReadAll(responses[1].Body)
	assert.Equal(t, "second", string(secondBody))
}

func TestUnpackResponseShouldPreserveTheStatus(t *testing.T) {
	// given a valid response with two parts with different status codes
	first, second := aResponse("first"), aResponse("second")
	first.StatusCode = 404
	first.Status = "404 Not Found"

	fatResponse := aFatResponse(first, second)

	// when
	responses, _ := UnpackResponse(fatResponse)

	// then
	assert.Equal(t, 404, responses[0].StatusCode)
	assert.Equal(t, "404 Not Found", responses[0].Status)

	assert.Equal(t, 200, responses[1].StatusCode)
	assert.Equal(t, "200 OK", responses[1].Status)
}

func TestUnpackResponseShouldPreserveTheHeaders(t *testing.T) {
	// given a valid response with two parts with custom header
	first, second := aResponse("first"), aResponse("second")
	first.Header.Set("X-First", "1st")
	second.Header.Set("X-Second", "2nd")

	fatResponse := aFatResponse(first, second)

	// when
	responses, _ := UnpackResponse(fatResponse)

	// then
	assert.Equal(t, "1st", responses[0].Header.Get("X-First"))
	assert.Equal(t, "2nd", responses[1].Header.Get("X-Second"))
}

func aResponse(body string) *http.Response {
	resp := new(http.Response)
	resp.StatusCode = 200
	resp.Status = "200 OK"
	resp.Proto = "HTTP/1.0"
	resp.ProtoMajor = 1
	resp.ProtoMinor = 0
	if body != "" {
		resp.Body = ioutil.NopCloser(bytes.NewBufferString(body))
		resp.ContentLength = int64(len(body))
	}
	resp.Header = http.Header{}
	resp.Header.Set("Content-Type", "text/plain")
	return resp
}

func aFatResponse(responses ...*http.Response) *http.Response {
	var buf bytes.Buffer
	multipartWriter := multipart.NewWriter(&buf)

	mimeHeaders := textproto.MIMEHeader(make(map[string][]string))
	mimeHeaders.Set("Content-Type", "application/http")

	for _, resp := range responses {
		part, _ := multipartWriter.CreatePart(mimeHeaders)
		resp.Write(part)
	}
	multipartWriter.Close()

	resp := new(http.Response)
	resp.StatusCode = 200
	resp.Status = "200 OK"
	resp.Proto = "HTTP/1.0"
	resp.ProtoMajor = 1
	resp.ProtoMinor = 0
	resp.Body = ioutil.NopCloser(&buf)
	resp.ContentLength = int64(buf.Len())
	resp.Header = http.Header{}
	resp.Header.Set("Content-Type", mime.FormatMediaType("multipart/mixed", map[string]string{"boundary": multipartWriter.Boundary()}))

	return resp
}
