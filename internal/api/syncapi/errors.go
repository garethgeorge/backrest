package syncapi

import (
	"fmt"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
)

type SyncError struct {
	State   v1.SyncConnectionState
	Message error
}

var _ error = (*SyncError)(nil)

func (e *SyncError) Error() string {
	return fmt.Sprintf("%v: %s", e.State, e.Message.Error())
}

func (e *SyncError) Unwrap() error {
	return e.Message
}
