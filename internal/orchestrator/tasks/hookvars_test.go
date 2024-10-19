package tasks

import (
	"testing"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
)

func TestHookVarsEventName(t *testing.T) {
	// for every value of condition, check that the event name is correct
	values := v1.Hook_Condition(0).Descriptor().Values()
	for i := 0; i < values.Len(); i++ {
		condition := v1.Hook_Condition(values.Get(i).Number())
		if condition == v1.Hook_CONDITION_UNKNOWN {
			continue
		}

		vars := HookVars{}
		if vars.EventName(condition) == "unknown" {
			t.Errorf("unexpected event name for condition %v", condition)
		}
	}
}
