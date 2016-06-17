package httpbatcher

import (
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/net/context"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestValidatorRejectNonPOSTMethods(t *testing.T) {
	// given a validateBatchRequestHandler
	v := validateBatchRequestHandler(func(c context.Context, w http.ResponseWriter, r *http.Request) {
		// not expected to be called
		assert.Fail(t, "Next handler should not be called if validation fails")
	})

	// when I pass a non POST request
	for _, method := range httpMethods {
		if method != http.MethodPost {
			rr := httptest.NewRecorder()
			req, err := http.NewRequest(method, "localhost", nil)
			require.Nil(t, err)

			v.ServeHTTP(context.TODO(), rr, req)

			// then the error code is
			assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
		}
	}
}

func TestValidatorFailIfContentTypeMalformed(t *testing.T) {
	// given a validateBatchRequestHandler
	v := validateBatchRequestHandler(func(c context.Context, w http.ResponseWriter, r *http.Request) {
		// not expected to be called
		assert.Fail(t, "Next handler should not be called if validation fails")
	})

	// and a request with a maflormed content type
	req, err := http.NewRequest("POST", "localhost", nil)
	require.Nil(t, err)

	req.Header.Set("Content-Type", "")

	// when
	rr := httptest.NewRecorder()
	v.ServeHTTP(context.TODO(), rr, req)

	// then
	assert.Equal(t, http.StatusUnsupportedMediaType, rr.Code)
	assert.Equal(t, "Content-Type malformed", rr.Body.String())
}

func TestValidatorRejectWrongContentType(t *testing.T) {
	// given a validateBatchRequestHandler
	v := validateBatchRequestHandler(func(c context.Context, w http.ResponseWriter, r *http.Request) {
		// not expected to be called
		assert.Fail(t, "Next handler should not be called if validation fails")
	})

	// and a request with a wrong content type
	req, err := http.NewRequest("POST", "localhost", nil)
	require.Nil(t, err)

	req.Header.Set("Content-Type", "text/plain")

	// when
	rr := httptest.NewRecorder()
	v.ServeHTTP(context.TODO(), rr, req)

	// then
	assert.Equal(t, http.StatusUnsupportedMediaType, rr.Code)
	assert.Equal(t, "Content-Type must be multipart/mixed", rr.Body.String())
}

func TestValidatorRejectMultipartMixedWithoutBoundary(t *testing.T) {
	// given a validateBatchRequestHandler
	v := validateBatchRequestHandler(func(c context.Context, w http.ResponseWriter, r *http.Request) {
		// not expected to be called
		assert.Fail(t, "Next handler should not be called if validation fails")
	})

	// and a request with a wrong content type
	req, err := http.NewRequest("POST", "localhost", nil)
	require.Nil(t, err)

	req.Header.Set("Content-Type", "multipart/mixed")

	// when
	rr := httptest.NewRecorder()
	v.ServeHTTP(context.TODO(), rr, req)

	// then
	assert.Equal(t, http.StatusUnsupportedMediaType, rr.Code)
	assert.Equal(t, "Content-Type is missing boundary parameter", rr.Body.String())
}

func TestNextIsCalledOnSuccessfulValidation(t *testing.T) {
	// given a validateBatchRequestHandler
	actualArgs := struct {
		Context        context.Context
		ResponseWriter http.ResponseWriter
		Request        *http.Request
	}{}

	v := validateBatchRequestHandler(func(c context.Context, w http.ResponseWriter, r *http.Request) {
		actualArgs.Context = c
		actualArgs.ResponseWriter = w
		actualArgs.Request = r
	})

	// and a valid request
	req, err := http.NewRequest("POST", "localhost", nil)
	require.Nil(t, err)
	req.Header.Set("Content-Type", mime.FormatMediaType("multipart/mixed", map[string]string{"boundary": "wuqhfkndk"}))

	// when
	rr := httptest.NewRecorder()
	rr.WriteHeader(http.StatusOK)

	v.ServeHTTP(context.TODO(), rr, req)

	// then
	assert.Equal(t, req, actualArgs.Request)
	assert.Equal(t, rr, actualArgs.ResponseWriter)
	assert.Equal(t, "wuqhfkndk", actualArgs.Context.Value("boundary"))
}

type fakeExtractor struct {
	argBoundary string
	argBody     io.Reader
	requests    []*http.Request
	err         error
}

func (fe *fakeExtractor) Extract(boundary string, body io.Reader) ([]*http.Request, error) {
	fe.argBoundary = boundary
	fe.argBody = body

	if fe.err != nil {
		return nil, fe.err
	}

	return fe.requests, nil
}

func TestExtractPayloadSuccessful(t *testing.T) {
	// given a good extractor
	extractor := &fakeExtractor{
		requests: []*http.Request{&http.Request{Host: "first"}, &http.Request{Host: "second"}},
	}

	// and the next handler
	actualArgs := struct {
		Context        context.Context
		ResponseWriter http.ResponseWriter
		Request        *http.Request
	}{}

	next := contextHandlerFunc(func(c context.Context, w http.ResponseWriter, r *http.Request) {
		actualArgs.Context = c
		actualArgs.ResponseWriter = w
		actualArgs.Request = r
	})

	// and some fake args
	c := context.WithValue(nil, "boundary", "qwjneofnsanf")

	w := httptest.NewRecorder()
	w.Header().Set("Say", "hello")

	r := &http.Request{Body: ioutil.NopCloser(strings.NewReader("bodycontent"))}

	// when
	chf := extractPayloadHandler(extractor, next)

	chf.ServeHTTP(c, w, r)

	// then
	assert.Equal(t, w, actualArgs.ResponseWriter)
	assert.Equal(t, r, actualArgs.Request)

	assert.Equal(t, "qwjneofnsanf", extractor.argBoundary)
	assert.Equal(t, r.Body, extractor.argBody)

	expectedContext := context.WithValue(c, "payload", extractor.requests)
	assert.Equal(t, expectedContext, actualArgs.Context)
}

func TestExtractPayload500Failure(t *testing.T) {
	// given a failing extractor
	extractor := &fakeExtractor{
		err: fmt.Errorf("Some error"),
	}

	// and the next handler
	next := contextHandlerFunc(func(c context.Context, w http.ResponseWriter, r *http.Request) {
		assert.Fail(t, "Next handler should not be called")
	})

	// and some fake args
	c := context.WithValue(nil, "boundary", "qwjneofnsanf")

	w := httptest.NewRecorder()
	w.Header().Set("Say", "hello")

	r := &http.Request{Body: ioutil.NopCloser(strings.NewReader("bodycontent"))}

	// when
	chf := extractPayloadHandler(extractor, next)

	chf.ServeHTTP(c, w, r)

	// then
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
