package hook

import (
	"errors"
	"testing"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
)

// TestApplyHookErrorPolicy tests that applyHookErrorPolicy is defined for all values of Hook_OnError.
func TestApplyHookErrorPolicy(t *testing.T) {
	values := v1.Hook_OnError(0).Descriptor().Values()
	for i := 0; i < values.Len(); i++ {
		applyHookErrorPolicy(v1.Hook_OnError(values.Get(i).Number()), errors.New("an error"))
	}
}
