package httpbatcher

import (
	"bufio"
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

func TestResponsePackerFunc(t *testing.T) {
	args := struct {
		responses []*http.Response
	}{}

	var f = func(responses []*http.Response) (boundary string, body io.Reader, err error) {
		args.responses = responses
		return "foo", strings.NewReader("bar"), fmt.Errorf("error msg")
	}

	rpf := responsePackerFunc(f)
	responses := someResponses(3)
	boundary, body, err := rpf.Pack(responses)
	bodyBytes, _ := ioutil.ReadAll(body)

	assert.Equal(t, responses, args.responses)
	assert.Equal(t, "foo", boundary)
	assert.Equal(t, "bar", string(bodyBytes))
	assert.EqualError(t, err, "error msg")
}

func TestSimpleResponsePacker(t *testing.T) {
	resps := someResponses(3)

	boundary, body, err := simpleResponsePacker(resps)
	require.Nil(t, err)

	assert.NotEmpty(t, boundary)
	require.NotNil(t, body)

	mr := multipart.NewReader(body, boundary)

	for i := 0; i < 3; i++ {
		part, erri := mr.NextPart()
		require.Nil(t, erri)

		actualResp, erri := http.ReadResponse(bufio.NewReader(part), nil)
		require.Nil(t, erri)

		mediaType, _, _ := mime.ParseMediaType(part.Header.Get("Content-Type"))
		assert.Equal(t, "application/http", mediaType)

		expectedResp := makeResponse(i)
		assert.Equal(t, expectedResp.Status, actualResp.Status)
		assert.Equal(t, expectedResp.StatusCode, actualResp.StatusCode)
		assert.Equal(t, expectedResp.Proto, actualResp.Proto)
		assert.Equal(t, expectedResp.ProtoMajor, actualResp.ProtoMajor)
		assert.Equal(t, expectedResp.ProtoMajor, actualResp.ProtoMajor)
		assert.Equal(t, expectedResp.ProtoMinor, actualResp.ProtoMinor)
		assert.Equal(t, expectedResp.Header, actualResp.Header)

		actualBody, _ := ioutil.ReadAll(actualResp.Body)
		expectedBody, _ := ioutil.ReadAll(expectedResp.Body)
		assert.Equal(t, expectedBody, actualBody)
	}

	_, err = mr.NextPart()
	assert.Equal(t, io.EOF, err)
}

func someResponses(n int) []*http.Response {
	resps := make([]*http.Response, n)
	for i := 0; i < n; i++ {
		resps[i] = makeResponse(i)
	}

	return resps
}

func makeResponse(i int) *http.Response {
	r := &http.Response{
		Status:     fmt.Sprintf("%d Blabla", 200+i),
		StatusCode: 200 + i,
		Proto:      "HTTP/1.0",
		ProtoMajor: 1,
		ProtoMinor: 0,
		Header:     http.Header{},
		Body:       ioutil.NopCloser(strings.NewReader(fmt.Sprintf("response-%d", i))),
	}
	r.Header.Set("Test-Header", fmt.Sprintf("value-%d", i))

	return r
}
