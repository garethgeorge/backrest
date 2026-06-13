package health

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/config"
	"github.com/stretchr/testify/assert"
)


type mockPinger struct {
	err error
}

func (m *mockPinger) PingContext(ctx context.Context) error {
	return m.err
}

type mockConfigStore struct {
	err error
}

func (m *mockConfigStore) Get() (*v1.Config, error) {
	if m.err != nil {
		return nil, m.err
	}

	return config.NewDefaultConfig(), nil
}

func (m *mockConfigStore) Update(config *v1.Config) error                              { return nil }
func (m *mockConfigStore) Transform(fn func(cfg *v1.Config) (*v1.Config, error)) error { return nil }


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
		configStoreErr error
		dbPingErr      error
		expectedCode   int
		expectedStatus string
	}{
		{
			name:           "Happy - App is ready",
			configStoreErr: nil,
			dbPingErr:      nil,
			expectedCode:   http.StatusOK,
			expectedStatus: `"READY"`,
		},
		{
			name:           "Sad - Config invalid or missing",
			configStoreErr: errors.New("simulated disk error"),
			dbPingErr:      nil,
			expectedCode:   http.StatusServiceUnavailable,
			expectedStatus: `"DOWN"`,
		},
		{
			name:           "Sad - Database locked or timed out",
			configStoreErr: nil,
			dbPingErr:      context.DeadlineExceeded,
			expectedCode:   http.StatusServiceUnavailable,
			expectedStatus: `"DOWN"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockConfigStore{err: tt.configStoreErr}

			configMgr := &config.ConfigManager{Store: store}
			db := &mockPinger{err: tt.dbPingErr}

			req, err := http.NewRequest("GET", "/readyz", nil)
			assert.NoError(t, err)

			rr := httptest.NewRecorder()
			handler := ReadyHandler(configMgr, db)

			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedCode, rr.Code, "HTTP Status code mismatch")
			assert.Contains(t, rr.Body.String(), tt.expectedStatus, "JSON body status mismatch")
		})
	}
}
