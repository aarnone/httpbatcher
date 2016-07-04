package httpbatcher

import (
	"bufio"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
)

type requestExtractor interface {
	Extract(boundary string, body io.Reader) ([]*http.Request, error)
}

type requestExtractorFunc func(string, io.Reader) ([]*http.Request, error)

func (ref requestExtractorFunc) Extract(boundary string, body io.Reader) ([]*http.Request, error) {
	return ref(boundary, body)
}

func simpleRequestExtractor(boundary string, reader io.Reader) ([]*http.Request, error) {
	var requests []*http.Request
	mpr := multipart.NewReader(reader, boundary)
	for {
		part, err := mpr.NextPart()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("can't parse the multipart body: %v", err)
		}

		partReq, err := http.ReadRequest(bufio.NewReader(part))
		if err != nil {
			return nil, fmt.Errorf("can't parse the multipart body: %v", err)
		}

		// RequestURI is allowed only for inbound requests
		partReq.RequestURI = ""

		requests = append(requests, partReq)
	}

	return requests, nil
}
