package repo

import (
	"fmt"
	"strings"
)

// TagForPlan returns a tag for the plan.
func TagForPlan(planId string) string {
	return fmt.Sprintf("plan:%s", planId)
}

// TagForInstance returns a tag for the instance.
func TagForInstance(instanceId string) string {
	return fmt.Sprintf("created-by:%s", instanceId)
}

// InstanceIDFromTags returns the instance ID from the tags, or an empty string if not found.
func InstanceIDFromTags(tags []string) string {
	for _, tag := range tags {
		if strings.HasPrefix(tag, "created-by:") {
			return tag[len("created-by:"):]
		}
	}
	return ""
}

// PlanFromTags returns the plan ID from the tags, or an empty string if not found.
func PlanFromTags(tags []string) string {
	for _, tag := range tags {
		if strings.HasPrefix(tag, "plan:") {
			return tag[len("plan:"):]
		}
	}
	return ""
}
