package protoutil

import (
	"strings"
	"testing"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
)

func TestStartConditionsMap(t *testing.T) {
	// Test that all conditions with "_START" in their name are correctly identified by IsStartCondition
	for cond := range v1.Hook_Condition_name {
		condEnum := v1.Hook_Condition(cond)
		condName := condEnum.String()
		if strings.Contains(condName, "_START") {
			if !IsStartCondition(condEnum) {
				t.Errorf("Condition %s contains '_START' but IsStartCondition returned false", condName)
			}
		} else {
			if IsStartCondition(condEnum) {
				t.Errorf("Condition %s does not contain '_START' but IsStartCondition returned true", condName)
			}
		}
	}
}

func TestErrorConditionsMap(t *testing.T) {
	// Special case for CONDITION_UNKNOWN which should be identified as an error condition
	if !IsErrorCondition(v1.Hook_CONDITION_UNKNOWN) {
		t.Errorf("CONDITION_UNKNOWN should be identified as an error condition")
	}

	// Special case for ANY_ERROR which should be identified as an error condition
	if !IsErrorCondition(v1.Hook_CONDITION_ANY_ERROR) {
		t.Errorf("CONDITION_ANY_ERROR should be identified as an error condition")
	}

	// Test that all conditions with "_ERROR" in their name are correctly identified by IsErrorCondition
	for cond := range v1.Hook_Condition_name {
		condEnum := v1.Hook_Condition(cond)
		condName := condEnum.String()

		// Skip the special cases we already checked
		if condEnum == v1.Hook_CONDITION_UNKNOWN || condEnum == v1.Hook_CONDITION_ANY_ERROR {
			continue
		}

		if strings.Contains(condName, "_ERROR") {
			if !IsErrorCondition(condEnum) {
				t.Errorf("Condition %s contains '_ERROR' but IsErrorCondition returned false", condName)
			}
		} else if IsErrorCondition(condEnum) {
			t.Errorf("Condition %s does not contain '_ERROR' but IsErrorCondition returned true", condName)
		}
	}
}

func TestSuccessConditionsMap(t *testing.T) {
	// Test that all conditions with "_SUCCESS" in their name are correctly identified by IsSuccessCondition
	for cond := range v1.Hook_Condition_name {
		condEnum := v1.Hook_Condition(cond)
		condName := condEnum.String()
		if strings.Contains(condName, "_SUCCESS") {
			if !IsSuccessCondition(condEnum) {
				t.Errorf("Condition %s contains '_SUCCESS' but IsSuccessCondition returned false", condName)
			}
		} else {
			if IsSuccessCondition(condEnum) {
				t.Errorf("Condition %s does not contain '_SUCCESS' but IsSuccessCondition returned true", condName)
			}
		}
	}
}
