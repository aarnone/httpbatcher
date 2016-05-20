package split

import "net/http"

// UnpackRequests split the multipart requests in its sub-requests
func UnpackRequests(r *http.Request) ([]*http.Request, error) {

	return make([]*http.Request, 0), nil
}
