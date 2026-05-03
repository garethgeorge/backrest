package syncapi

import (
	"fmt"

	"github.com/garethgeorge/backrest/gen/go/v1sync"
)

type SyncError struct {
	State   v1sync.ConnectionState
	Message error
}

var _ error = (*SyncError)(nil)

func (e *SyncError) Error() string {
	return fmt.Sprintf("syncerror %v: %s", e.State, e.Message.Error())
}

func (e *SyncError) Unwrap() error {
	return e.Message
}

func NewSyncErrorDisconnected(message error) *SyncError {
	return &SyncError{
		State:   v1sync.ConnectionState_CONNECTION_STATE_DISCONNECTED,
		Message: message,
	}
}

func NewSyncErrorAuth(message error) *SyncError {
	return &SyncError{
		State:   v1sync.ConnectionState_CONNECTION_STATE_ERROR_AUTH,
		Message: message,
	}
}

func NewSyncErrorProtocol(message error) *SyncError {
	return &SyncError{
		State:   v1sync.ConnectionState_CONNECTION_STATE_ERROR_PROTOCOL,
		Message: message,
	}
}

func NewSyncErrorInternal(message error) *SyncError {
	return &SyncError{
		State:   v1sync.ConnectionState_CONNECTION_STATE_ERROR_INTERNAL,
		Message: message,
	}
}
