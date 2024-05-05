package stringutil

import "regexp"

var (
	sanitizeIDRegex = regexp.MustCompile(`[^a-zA-Z0-9_\-\.]+`) // matches invalid characters in an ID
	idRegex         = regexp.MustCompile(`[a-zA-Z0-9_\-\.]*`)  // matches a valid ID (including empty string)
)

func SanitizeID(id string) string {
	return sanitizeIDRegex.ReplaceAllString(id, "_")
}

func ValidateID(id string) bool {
	return idRegex.MatchString(id)
}
