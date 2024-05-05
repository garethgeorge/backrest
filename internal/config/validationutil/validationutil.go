package validationutil

import (
	"errors"
	"fmt"
	"regexp"
)

var (
	IDMaxLen        = 50                                       // maximum length of an ID
	sanitizeIDRegex = regexp.MustCompile(`[^a-zA-Z0-9_\-\.]+`) // matches invalid characters in an ID
	idRegex         = regexp.MustCompile(`[a-zA-Z0-9_\-\.]*`)  // matches a valid ID (including empty string)
)

func SanitizeID(id string) string {
	return sanitizeIDRegex.ReplaceAllString(id, "_")
}

// ValidateID checks if an ID is valid.
// It returns an error if the ID contains invalid characters, is empty, or is too long.
// The maxLen parameter is the maximum length of the ID. If maxLen is 0, the ID length is not checked.
func ValidateID(id string, maxLen int) error {
	if !idRegex.MatchString(id) {
		return errors.New("contains invalid characters")
	}
	if len(id) == 0 {
		return errors.New("empty")
	}
	if maxLen > 0 && len(id) > maxLen {
		return fmt.Errorf("too long (> %d chars)", maxLen)
	}
	return nil
}
