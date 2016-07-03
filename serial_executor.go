package httpbatcher

import (
	"fmt"
	"net/http"
)

// ExecuteSerially execute the requests with http.DefaultClient one after the other
func ExecuteSerially(requests []*http.Request) ([]*http.Response, error) {

	responses := make([]*http.Response, len(requests))
	for i := range requests {
		resp, err := http.DefaultClient.Do(requests[i])
		if err != nil {
			return nil, fmt.Errorf("error occurred while executing http request: %v", err)
		}

		responses[i] = resp
	}

	return responses, nil
}
