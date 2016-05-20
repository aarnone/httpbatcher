package split

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
)

// ErrMalformedMediaType is returned when the media type cannot be parsed
var ErrMalformedMediaType = fmt.Errorf("Malformed media type")

// ErrInvalidMediaType is returned when the media type is not multipart/mixed with boundary param set
var ErrInvalidMediaType = fmt.Errorf("Invalid media type, it should be multipart/mixed with boundary parameter")

// ErrMalformedRequest is returned when a part can be read into a http.Request
var ErrMalformedRequest = fmt.Errorf("Invalid request part format")

// UnpackRequests split the multipart requests in its sub-requests
func UnpackRequests(r *http.Request) ([]*http.Request, error) {
	delimiter, err := extractDelimiter(r)
	if err != nil {
		return nil, err
	}

	var requests []*http.Request
	multiReader := multipart.NewReader(r.Body, delimiter)
	for {
		part, err := multiReader.NextPart()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		partReq, err := http.ReadRequest(bufio.NewReader(part))
		if err != nil {
			log.Printf(err.Error())
			return nil, ErrMalformedRequest
		}
		fixRequestForClientCall(partReq)

		requests = append(requests, partReq)
	}

	return requests, nil
}

func extractDelimiter(r *http.Request) (string, error) {
	mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))

	if err != nil {
		return "", ErrMalformedMediaType
	}
	if mediaType != "multipart/mixed" {
		return "", ErrInvalidMediaType
	}

	boundary, ok := params["boundary"]
	if !ok {
		return "", ErrInvalidMediaType
	}
	return boundary, nil
}

func fixRequestForClientCall(r *http.Request) {
	r.RequestURI = ""
	r.URL.Host = r.Host

	if r.URL.Scheme == "" {
		r.URL.Scheme = "http"
	}
}
