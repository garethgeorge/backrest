package protoutil

import (
	v1 "github.com/garethgeorge/backrest/gen/go/v1"
)

var startConditionsMap = map[v1.Hook_Condition]bool{
	v1.Hook_CONDITION_CHECK_START:    true,
	v1.Hook_CONDITION_PRUNE_START:    true,
	v1.Hook_CONDITION_SNAPSHOT_START: true,
}

var errorConditionsMap = map[v1.Hook_Condition]bool{
	v1.Hook_CONDITION_ANY_ERROR:      true,
	v1.Hook_CONDITION_CHECK_ERROR:    true,
	v1.Hook_CONDITION_PRUNE_ERROR:    true,
	v1.Hook_CONDITION_SNAPSHOT_ERROR: true,
	v1.Hook_CONDITION_UNKNOWN:        true,
}

var logConditionsMap = map[v1.Hook_Condition]bool{
	v1.Hook_CONDITION_SNAPSHOT_END: true,
}

var successConditionsMap = map[v1.Hook_Condition]bool{
	v1.Hook_CONDITION_CHECK_SUCCESS:    true,
	v1.Hook_CONDITION_PRUNE_SUCCESS:    true,
	v1.Hook_CONDITION_SNAPSHOT_SUCCESS: true,
}

// IsErrorCondition returns true if the event is an error condition.
func IsErrorCondition(event v1.Hook_Condition) bool {
	return errorConditionsMap[event]
}

// IsLogCondition returns true if the event is a log condition.
func IsLogCondition(event v1.Hook_Condition) bool {
	return logConditionsMap[event]
}

// IsStartCondition returns true if the event is a start condition.
func IsStartCondition(event v1.Hook_Condition) bool {
	return startConditionsMap[event]
}

// IsSuccessCondition returns true if the event is a success condition.
func IsSuccessCondition(event v1.Hook_Condition) bool {
	return successConditionsMap[event]
}
