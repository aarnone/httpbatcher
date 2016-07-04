package httpbatcher

import (
	"io"
	"log"
	"mime"
	"net/http"

	"golang.org/x/net/context"
)

// RequestExecutor represent the execution of a set of http.Requests independently by the scheduling strategy
type RequestExecutor interface {
	// Execute the requests and return the corresponding responses
	Execute([]*http.Request) ([]*http.Response, error)
}

type requestExecutorFunc func([]*http.Request) ([]*http.Response, error)

func (ref requestExecutorFunc) Execute(requests []*http.Request) ([]*http.Response, error) {
	return ref(requests)
}

// New instatiate an handler to manage http batch requests
func New() http.Handler {
	requestExtractor := requestExtractorFunc(simpleRequestExtractor)
	reqestExecutor := requestExecutorFunc(ExecuteSerially)
	responsePacker := responsePackerFunc(simpleResponsePacker)
	return initHTTPHandler(
		validateBatchRequestHandler(
			extractPayloadHandler(requestExtractor,
				executeRequestsHandler(reqestExecutor,
					packResponsesHandler(responsePacker, nil)))))
}

func validateBatchRequestHandler(next contextHandlerFunc) contextHandlerFunc {
	return func(c context.Context, w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}

		mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil {
			w.WriteHeader(http.StatusUnsupportedMediaType)
			w.Write([]byte("Content-Type malformed"))
			return
		}
		if mediaType != "multipart/mixed" {
			w.WriteHeader(http.StatusUnsupportedMediaType)
			w.Write([]byte("Content-Type must be multipart/mixed"))
			return
		}

		boundary := params["boundary"]
		if boundary == "" {
			w.WriteHeader(http.StatusUnsupportedMediaType)
			w.Write([]byte("Content-Type is missing boundary parameter"))
			return
		}

		next(context.WithValue(c, "boundary", boundary), w, r)
	}
}

func extractPayloadHandler(extractor requestExtractor, next contextHandlerFunc) contextHandlerFunc {
	return func(c context.Context, w http.ResponseWriter, r *http.Request) {
		requests, err := extractor.Extract(c.Value("boundary").(string), r.Body)
		if err != nil {
			log.Println("Requests extraction failed")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		next(context.WithValue(c, "payload", requests), w, r)
	}
}

func executeRequestsHandler(executor RequestExecutor, next contextHandlerFunc) contextHandlerFunc {
	return func(c context.Context, w http.ResponseWriter, r *http.Request) {
		requests := c.Value("payload")

		responses, err := executor.Execute(requests.([]*http.Request))
		if err != nil {
			log.Printf("Request execution failed: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		next(context.WithValue(c, "responses", responses), w, r)
	}
}

func packResponsesHandler(packer responsePacker, next contextHandlerFunc) contextHandlerFunc {
	return func(c context.Context, w http.ResponseWriter, r *http.Request) {
		responses := c.Value("responses").([]*http.Response)

		boundary, body, err := packer.Pack(responses)
		if err != nil {
			log.Println("Response packing failed")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		mediaType := mime.FormatMediaType("multipart/mixed", map[string]string{"boundary": boundary})

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", mediaType)

		io.Copy(w, body)
	}
}
