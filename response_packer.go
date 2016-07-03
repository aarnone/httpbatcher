package httpbatcher

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
)

type responsePacker interface {
	Pack(responses []*http.Response) (boundary string, body io.Reader, err error)
}

type responsePackerFunc func(responses []*http.Response) (boundary string, body io.Reader, err error)

func (rpf responsePackerFunc) Pack(responses []*http.Response) (boundary string, body io.Reader, err error) {
	return rpf(responses)
}

func simpleResponsePacker(responses []*http.Response) (boundary string, body io.Reader, err error) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	for _, resp := range responses {
		headers := textproto.MIMEHeader{}
		headers.Set("Content-Type", "application/http")

		bodyWriter, err := mw.CreatePart(headers)
		if err != nil {
			return "", nil, fmt.Errorf("Impossibile to create a multipart body: %v", err)
		}

		resp.Write(bodyWriter)
	}

	mw.Close()

	return mw.Boundary(), &buf, nil
}
