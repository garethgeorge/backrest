package health

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// LivenessHandler is a simple HTTP handler for liveness checks.
// It responds with HTTP 200 OK and the body "OK".
func LivenessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK\n"))
}

type Pinger interface {
	PingContext(ctx context.Context) error
}

type ReadyResponse struct {
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

func ReadyHandler(db Pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Check if database is available with a timeout
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		if err := db.PingContext(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable) // 503
			json.NewEncoder(w).Encode(ReadyResponse{
				Status: "DOWN",
				Reason: "Database is locked or unresponsive",
			})
			return
		}

		w.WriteHeader(http.StatusOK) // 200
		json.NewEncoder(w).Encode(ReadyResponse{
			Status: "READY",
		})
	}
}
