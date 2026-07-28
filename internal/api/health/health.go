package health

import (
	"encoding/json"
	"net/http"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/oplog"
)

const readyTimeout = 2 * time.Second

// LivenessHandler responds 200 OK. 
func LivenessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK\n"))
}

type ReadyResponse struct {
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

func writeReady(w http.ResponseWriter, status, reason string) {
	code := http.StatusOK
	if status != "READY" {
		code = http.StatusServiceUnavailable
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(ReadyResponse{Status: status, Reason: reason})
}

// ReadyHandler returns a handler for /readyz. It checks that the
// operation log is queryable by running a simple query.
func ReadyHandler(opLog *oplog.OpLog) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		done := make(chan error, 1)
		q := oplog.Query{}.SetReversed(true).SetLimit(1)

		// Run the query in a goroutine so we can enforce our own timeout.
		// OpLog.Query does not accept a context, so a stuck SQLite call
		// would hang the handler indefinitely without this.
		go func() {
			done <- opLog.Query(q, func(op *v1.Operation) error {
				return nil
			})
		}()

		select {
		case err := <-done:
			if err != nil {
				writeReady(w, "DOWN", "operation log query failed: "+err.Error())
				return
			}
		case <-time.After(readyTimeout):
			writeReady(w, "DOWN", "operation log query timed out")
			return
		}

		writeReady(w, "READY", "")
	}
}
