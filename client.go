package httpbatcher

import (
	"bytes"
	"fmt"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
)

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
