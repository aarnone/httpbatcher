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

func TestExtractPayloadHandlerSuccessful(t *testing.T) {
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

func TestExtractPayloadHandler500Failure(t *testing.T) {
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

type fakeExecutor struct {
	argRequests []*http.Request
	responses   []*http.Response
	err         error
}

func (fe *fakeExecutor) Execute(requests []*http.Request) ([]*http.Response, error) {
	fe.argRequests = requests
	if fe.err != nil {
		return nil, fe.err
	}

	return fe.responses, nil
}

func TestExecuteRequestsHandlerSuccessful(t *testing.T) {
	// given a good executor
	executor := &fakeExecutor{
		responses: []*http.Response{&http.Response{Status: "first"}, &http.Response{Status: "second"}},
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
	requests := []*http.Request{&http.Request{Host: "one"}, &http.Request{Host: "two"}}
	c := context.WithValue(nil, "payload", requests)

	w := httptest.NewRecorder()
	w.Header().Set("Say", "hello")

	r := &http.Request{Body: ioutil.NopCloser(strings.NewReader("bodycontent"))}

	// when
	chf := executeRequestsHandler(executor, next)

	chf.ServeHTTP(c, w, r)

	// then
	assert.Equal(t, w, actualArgs.ResponseWriter)
	assert.Equal(t, r, actualArgs.Request)

	assert.Equal(t, requests, executor.argRequests)

	expectedContext := context.WithValue(c, "responses", executor.responses)
	assert.Equal(t, expectedContext, actualArgs.Context)
}

func TestExecuteRequestsHandler500Failure(t *testing.T) {
	// given a failing executor
	executor := &fakeExecutor{
		err: fmt.Errorf("Some generic error"),
	}

	// and the next handler
	next := contextHandlerFunc(func(c context.Context, w http.ResponseWriter, r *http.Request) {
		assert.Fail(t, "Next handler should not be called")
	})

	// and some fake args
	requests := []*http.Request{&http.Request{Host: "one"}, &http.Request{Host: "two"}}
	c := context.WithValue(nil, "payload", requests)

	w := httptest.NewRecorder()
	w.Header().Set("Say", "hello")

	r := &http.Request{Body: ioutil.NopCloser(strings.NewReader("bodycontent"))}

	// when
	chf := executeRequestsHandler(executor, next)

	chf.ServeHTTP(c, w, r)

	// then
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

type fakePacker struct {
	argResponses []*http.Response
	err          error
	body         io.Reader
	boundary     string
}

func (fp *fakePacker) Pack(responses []*http.Response) (string, io.Reader, error) {
	fp.argResponses = responses

	if fp.err != nil {
		return "", nil, fp.err
	}

	return fp.boundary, fp.body, nil
}

func TestPackResponsesHandlerSuccessful(t *testing.T) {
	// given a good packer
	packer := &fakePacker{
		boundary: "qjfoiwenienf",
		body:     strings.NewReader("bodybodybody"),
	}

	// and the next handler
	next := contextHandlerFunc(func(c context.Context, w http.ResponseWriter, r *http.Request) {
		assert.Fail(t, "Next handler should not be called")
	})

	// and some fake args
	responses := []*http.Response{&http.Response{Status: "one"}, &http.Response{Status: "two"}}
	c := context.WithValue(nil, "responses", responses)
	r := &http.Request{Body: ioutil.NopCloser(strings.NewReader("bodycontent"))}

	// when
	w := httptest.NewRecorder()
	chf := packResponsesHandler(packer, next)

	chf.ServeHTTP(c, w, r)

	// then
	assert.Equal(t, w.Code, http.StatusOK)

	mediaType, params, err := mime.ParseMediaType(w.Header().Get("Content-Type"))
	require.Nil(t, err)

	assert.Equal(t, "multipart/mixed", mediaType)
	assert.Equal(t, "qjfoiwenienf", params["boundary"])

	assert.Equal(t, "bodybodybody", w.Body.String())
}

func TestPackResponsesHandler500Failure(t *testing.T) {
	// given a failing packer
	packer := &fakePacker{
		err: fmt.Errorf("Some generic error"),
	}

	// and the next handler
	next := contextHandlerFunc(func(c context.Context, w http.ResponseWriter, r *http.Request) {
		assert.Fail(t, "Next handler should not be called")
	})

	// and some fake args
	responses := []*http.Response{&http.Response{Status: "one"}, &http.Response{Status: "two"}}
	c := context.WithValue(nil, "responses", responses)
	r := &http.Request{Body: ioutil.NopCloser(strings.NewReader("bodycontent"))}

	// when
	w := httptest.NewRecorder()
	chf := packResponsesHandler(packer, next)

	chf.ServeHTTP(c, w, r)

	// then
	assert.Equal(t, w.Code, http.StatusInternalServerError)
}

func TestRequestExecutorFunc(t *testing.T) {
	var args = struct{ requests []*http.Request }{}

	executor := requestExecutorFunc(func(requests []*http.Request) ([]*http.Response, error) {
		args.requests = requests
		return someResponses(3), fmt.Errorf("some error")
	})

	requests := someRequests(3)
	actualResponses, err := executor.Execute(requests)

	assert.Equal(t, requests, args.requests)
	assert.EqualError(t, err, "some error")
	assert.Equal(t, actualResponses, someResponses(3))

}
