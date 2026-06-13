package health

import (
	"net/http"
)

// Handler is a simple HTTP handler for health checks.
// It responds with HTTP 200 OK and the body "OK".
func Handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK\n"))
}
