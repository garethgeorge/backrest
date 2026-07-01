package health

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)


type mockPinger struct {
	err error
}

func (m *mockPinger) PingContext(ctx context.Context) error {
	return m.err
}


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
	tests := []struct {
		name           string
		dbPingErr      error
		expectedCode   int
		expectedStatus string
	}{
		{
			name:           "Happy - App is ready",
			dbPingErr:      nil,
			expectedCode:   http.StatusOK,
			expectedStatus: `"READY"`,
		},
		{
			name:           "Sad - Database locked or timed out",
			dbPingErr:      context.DeadlineExceeded,
			expectedCode:   http.StatusServiceUnavailable,
			expectedStatus: `"DOWN"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &mockPinger{err: tt.dbPingErr}

			req, err := http.NewRequest("GET", "/readyz", nil)
			assert.NoError(t, err)

			rr := httptest.NewRecorder()
			handler := ReadyHandler(db)

			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedCode, rr.Code, "HTTP Status code mismatch")
			assert.Contains(t, rr.Body.String(), tt.expectedStatus, "JSON body status mismatch")
		})
	}
}
