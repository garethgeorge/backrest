package stringutil

import "regexp"

func SanitizeID(id string) string {
	reg := regexp.MustCompile(`[^a-zA-Z0-9_\-\.]+`)
	return reg.ReplaceAllString(id, "_")
}
