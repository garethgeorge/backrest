package tasks

import (
	"errors"
	"testing"
	"time"

	"github.com/garethgeorge/backrest/pkg/restic"
)

func TestIsNetworkError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "restic ErrRepoNotFound",
			err:      restic.ErrRepoNotFound,
			expected: true,
		},
		{
			name:     "network path not found error",
			err:      errors.New("CreateFile \\\\NAS\\User\\Backup\\config: The network path was not found."),
			expected: true,
		},
		{
			name:     "no such host error",
			err:      errors.New("dial tcp: lookup gotify.mydomain.com: no such host"),
			expected: true,
		},
		{
			name:     "connection refused error",
			err:      errors.New("dial tcp 192.168.1.1:22: connection refused"),
			expected: true,
		},
		{
			name:     "connection timed out error",
			err:      errors.New("dial tcp 192.168.1.1:22: connection timed out"),
			expected: true,
		},
		{
			name:     "network unreachable error",
			err:      errors.New("dial tcp 192.168.1.1:22: network is unreachable"),
			expected: true,
		},
		{
			name:     "host is down error",
			err:      errors.New("dial tcp 192.168.1.1:22: host is down"),
			expected: true,
		},
		{
			name:     "no route to host error",
			err:      errors.New("dial tcp 192.168.1.1:22: no route to host"),
			expected: true,
		},
		{
			name:     "i/o timeout error",
			err:      errors.New("dial tcp 192.168.1.1:22: i/o timeout"),
			expected: true,
		},
		{
			name:     "connection reset error",
			err:      errors.New("read tcp 192.168.1.1:22: connection reset by peer"),
			expected: true,
		},
		{
			name:     "wrapped restic ErrRepoNotFound",
			err:      errors.New("failed to access repository: " + restic.ErrRepoNotFound.Error()),
			expected: false, // wrapped errors need errors.Is() to be detected properly
		},
		{
			name:     "non-network error",
			err:      errors.New("file not found"),
			expected: false,
		},
		{
			name:     "authentication error",
			err:      errors.New("permission denied"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNetworkError(tt.err)
			if result != tt.expected {
				t.Errorf("isNetworkError(%v) = %v, expected %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestNetworkRetryBackoffPolicy(t *testing.T) {
	tests := []struct {
		name            string
		attempt         int
		expectedMinimum time.Duration
		expectedMaximum time.Duration
	}{
		{
			name:            "first retry (attempt 0)",
			attempt:         0,
			expectedMinimum: 2 * time.Second,
			expectedMaximum: 2 * time.Second,
		},
		{
			name:            "second retry (attempt 1)",
			attempt:         1,
			expectedMinimum: 4 * time.Second,
			expectedMaximum: 4 * time.Second,
		},
		{
			name:            "third retry (attempt 2)",
			attempt:         2,
			expectedMinimum: 8 * time.Second,
			expectedMaximum: 8 * time.Second,
		},
		{
			name:            "fourth retry (attempt 3)",
			attempt:         3,
			expectedMinimum: 16 * time.Second,
			expectedMaximum: 16 * time.Second,
		},
		{
			name:            "fifth retry (attempt 4)",
			attempt:         4,
			expectedMinimum: 30 * time.Second, // should be capped at 30s
			expectedMaximum: 30 * time.Second,
		},
		{
			name:            "many retries (attempt 10)",
			attempt:         10,
			expectedMinimum: 30 * time.Second, // should be capped at 30s
			expectedMaximum: 30 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := networkRetryBackoffPolicy(tt.attempt)
			if result < tt.expectedMinimum || result > tt.expectedMaximum {
				t.Errorf("networkRetryBackoffPolicy(%d) = %v, expected between %v and %v", 
					tt.attempt, result, tt.expectedMinimum, tt.expectedMaximum)
			}
		})
	}
}

func TestIsNetworkErrorWithWrappedErrors(t *testing.T) {
	// Test with properly wrapped restic.ErrRepoNotFound
	wrappedErr := errors.New("repository access failed: " + restic.ErrRepoNotFound.Error())
	if isNetworkError(wrappedErr) {
		t.Errorf("isNetworkError should not detect wrapped ErrRepoNotFound in plain string, got true")
	}

	// Test with errors.Is compatible wrapping
	properlyWrappedErr := errors.Join(errors.New("repository access failed"), restic.ErrRepoNotFound)
	if !isNetworkError(properlyWrappedErr) {
		t.Errorf("isNetworkError should detect properly wrapped ErrRepoNotFound, got false")
	}
}

func TestBackupNetworkRetryBehavior(t *testing.T) {
	// Test that network errors return TaskRetryError
	networkErrors := []error{
		restic.ErrRepoNotFound,
		errors.New("CreateFile \\\\NAS\\User\\Backup\\config: The network path was not found."),
		errors.New("dial tcp: lookup server.example.com: no such host"),
	}

	for _, netErr := range networkErrors {
		t.Run("network error: "+netErr.Error(), func(t *testing.T) {
			if !isNetworkError(netErr) {
				t.Fatalf("Expected network error to be detected")
			}

			// Create a TaskRetryError as the backup task would
			retryErr := &TaskRetryError{
				Err:     netErr,
				Backoff: networkRetryBackoffPolicy,
			}

			// Verify the retry error has the correct properties
			if retryErr.Err != netErr {
				t.Errorf("Expected wrapped error to be preserved")
			}

			// Test backoff behavior
			delay1 := retryErr.Backoff(0)
			delay2 := retryErr.Backoff(1)
			if delay1 >= delay2 {
				t.Errorf("Expected exponential backoff, got delay1=%v, delay2=%v", delay1, delay2)
			}

			// Verify it starts with reasonable delay for network issues
			if delay1 != 2*time.Second {
				t.Errorf("Expected first retry delay of 2s, got %v", delay1)
			}
		})
	}
}