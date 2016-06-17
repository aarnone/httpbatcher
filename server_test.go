package httpbatcher

import (
	"mime"
	"net/http"
	"net/http/httptest"
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
	var nextCalled bool
	v := validateBatchRequestHandler(func(c context.Context, w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	// and a valid request
	req, err := http.NewRequest("POST", "localhost", nil)
	require.Nil(t, err)
	req.Header.Set("Content-Type", mime.FormatMediaType("multipart/mixed", map[string]string{"boundary": "wuqhfkndk"}))

	// when
	rr := httptest.NewRecorder()
	v.ServeHTTP(context.TODO(), rr, req)

	// then
	assert.True(t, nextCalled)
}
