package split

import (
	"bytes"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
)

// WriteResponses serialize the responses passed as argument into the ResponseWriter
func WriteResponses(w http.ResponseWriter, responses []*http.Response) error {
	var buf bytes.Buffer
	multipartWriter := multipart.NewWriter(&buf)

	mimeHeaders := textproto.MIMEHeader(make(map[string][]string))
	mimeHeaders.Set("Content-Type", "application/http")

	for _, resp := range responses {
		part, err := multipartWriter.CreatePart(mimeHeaders)
		if err != nil {
			return err
		}
		resp.Write(part)
	}

	multipartWriter.Close()

	w.Header().Set("Content-Type", mime.FormatMediaType("multipart/mixed", map[string]string{"boundary": multipartWriter.Boundary()}))
	w.WriteHeader(http.StatusOK)
	buf.WriteTo(w)
	return nil
}
