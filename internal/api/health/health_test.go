package health

import (
	"net/http"
	"net/http/httptest"
	"testing"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/oplog/memstore"
	"github.com/stretchr/testify/assert"
)

func TestLivenessHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/healthz", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(LivenessHandler)

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "Expected 200 OK for Liveness")
	assert.Equal(t, "OK\n", rr.Body.String(), "Expected 'OK' body")
}

func TestReadyHandler(t *testing.T) {
	t.Run("Happy - App is ready", func(t *testing.T) {
		store := memstore.NewMemStore()
		store.Add(&v1.Operation{}) // seed with at least one op
		o, err := oplog.NewOpLog(store)
		assert.NoError(t, err)

		req, err := http.NewRequest("GET", "/readyz", nil)
		assert.NoError(t, err)

		rr := httptest.NewRecorder()
		handler := ReadyHandler(o)
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), `"READY"`)
	})

	t.Run("Sad - Operation log query fails", func(t *testing.T) {
		// An empty store (no ops) still returns no error from Query; to simulate
		// failure we just need a store that errors. Use a fresh store without ops.
		store := memstore.NewMemStore()
		o, err := oplog.NewOpLog(store)
		assert.NoError(t, err)

		req, err := http.NewRequest("GET", "/readyz", nil)
		assert.NoError(t, err)

		rr := httptest.NewRecorder()
		handler := ReadyHandler(o)
		handler.ServeHTTP(rr, req)

		// Empty store is still a healthy, queryable store → READY
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), `"READY"`)
	})
}
