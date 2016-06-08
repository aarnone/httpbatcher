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
	"net/textproto"
)

// ErrInvalidMediaType is returned when the media type is not multipart/mixed
var ErrInvalidMediaType = fmt.Errorf("Expected media type multipart/mixed")

// ErrMultipartBoundaryNotDefined is returned when the media type is not multipart/mixed
var ErrMultipartBoundaryNotDefined = fmt.Errorf("Media type multipart/mixed require boundary parameter")

// BuildRequest encode a slice of requests into a single, multipart one
func BuildRequest(requests []*http.Request, targetURL string) (*http.Request, error) {

	var body bytes.Buffer
	req, err := http.NewRequest(http.MethodPost, targetURL, &body)
	if err != nil {
		return nil, fmt.Errorf("Impossible to create the wrapper request: %v", err)
	}

	multiWriter := multipart.NewWriter(&body)

	partHeaders := textproto.MIMEHeader{}
	partHeaders.Set("Content-Type", "application/http")
	for _, r := range requests {
		partBody, errFor := multiWriter.CreatePart(partHeaders)
		if errFor != nil {
			return nil, fmt.Errorf("Error while creating request part: %v", errFor)
		}

		r.WriteProxy(partBody)
	}
	multiWriter.Close()

	mediaType := mime.FormatMediaType("multipart/mixed", map[string]string{"boundary": multiWriter.Boundary()})
	req.Header.Set("Content-Type", mediaType)

	return req, err
}

// UnpackResponse decode a multipart response into a slice containing its parts
func UnpackResponse(fatResponse *http.Response) ([]*http.Response, error) {
	if fatResponse.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(fatResponse.Body)
		return nil, fmt.Errorf("Response status is %v: %v", fatResponse.StatusCode, string(body))
	}

	mediaType, params, err := mime.ParseMediaType(fatResponse.Header.Get("Content-Type"))
	if err != nil {
		return nil, fmt.Errorf("Can't parse the content type: %v", err)
	}
	if mediaType != "multipart/mixed" {
		return nil, ErrInvalidMediaType
	}
	if params["boundary"] == "" {
		return nil, ErrMultipartBoundaryNotDefined
	}

	mr := multipart.NewReader(fatResponse.Body, params["boundary"])
	var responses []*http.Response
	for {
		np, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		resp, err := http.ReadResponse(bufio.NewReader(np), nil)
		if err != nil {
			return nil, fmt.Errorf("Unexpected error: %v", err)
		}

		responses = append(responses, resp)
	}

	return responses, nil
}
