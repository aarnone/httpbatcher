package split

import "net/http"

// WriteResponses serialize the responses passed as argument into the ResponseWriter
func WriteResponses(w http.ResponseWriter, esponses []*http.Response) error {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Not impelmented yet"))
	return nil
}
