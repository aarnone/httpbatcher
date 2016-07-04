package httpbatcher

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestExtractorFunc(t *testing.T) {
	args := struct {
		boundary string
		reader   []byte
	}{}

	var f = func(boundary string, reader io.Reader) ([]*http.Request, error) {
		args.boundary = boundary
		args.reader, _ = ioutil.ReadAll(reader)
		return []*http.Request{&http.Request{Host: "foo"}}, fmt.Errorf("error msg")
	}

	ref := requestExtractorFunc(f)
	requests, err := ref.Extract("1234asd", strings.NewReader("qwerty"))

	assert.Equal(t, "1234asd", args.boundary)
	assert.Equal(t, "qwerty", string(args.reader))
	assert.EqualError(t, err, "error msg")
	assert.Equal(t, []*http.Request{&http.Request{Host: "foo"}}, requests)
}

func TestSimpleRequestExtractor(t *testing.T) {
	requests := someRequests(3)
	fatRequest, err := BuildRequest("localhost", requests...)
	require.Nil(t, err)

	_, params, err := mime.ParseMediaType(fatRequest.Header.Get("Content-Type"))
	require.Nil(t, err)

	actualRequests, err := simpleRequestExtractor(params["boundary"], fatRequest.Body)
	assert.Nil(t, err)

	for i := range requests {
		var buf bytes.Buffer
		requests[i].WriteProxy(&buf)
		expectedRequest, err := http.ReadRequest(bufio.NewReader(&buf))
		require.Nil(t, err)

		expectedRequest.RequestURI = "" // allowed only in incoming requests, not client requests
		assert.Equal(t, expectedRequest, actualRequests[i])
	}
}

func TestSimpleRequestExtractorBadContent(t *testing.T) {
	_, err := simpleRequestExtractor("xyz", strings.NewReader("badbody"))
	assert.EqualError(t, err, "can't parse the multipart body: multipart: NextPart: EOF")
}

func TestSimpleRequestExtractorBadPartBody(t *testing.T) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	p, e := w.CreatePart(nil)
	require.Nil(t, e)

	p.Write([]byte("this is not a valid request"))
	w.Close()

	_, err := simpleRequestExtractor(w.Boundary(), &buf)
	assert.EqualError(t, err, `can't parse the multipart body: malformed HTTP version "not a valid request"`)
}
