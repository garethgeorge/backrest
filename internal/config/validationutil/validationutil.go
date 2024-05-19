package validationutil

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	IDMaxLen        = 50                                       // maximum length of an ID
	sanitizeIDRegex = regexp.MustCompile(`[^a-zA-Z0-9_\-\.]+`) // matches invalid characters in an ID
	idRegex         = regexp.MustCompile(`[a-zA-Z0-9_\-\.]*`)  // matches a valid ID (including empty string)
)

var (
	ErrEmpty        = errors.New("empty")
	ErrTooLong      = errors.New("too long")
	ErrInvalidChars = errors.New("contains invalid characters")
)

func SanitizeID(id string) string {
	return sanitizeIDRegex.ReplaceAllString(id, "_")
}

// ValidateID checks if an ID is valid.
// It returns an error if the ID contains invalid characters, is empty, or is too long.
// The maxLen parameter is the maximum length of the ID. If maxLen is 0, the ID length is not checked.
func ValidateID(id string, maxLen int) error {
	if strings.HasPrefix(id, "__") && strings.HasSuffix(id, "__") {
		return errors.New("IDs starting and ending with '__' are reserved by backrest")
	}
	if !idRegex.MatchString(id) {
		return ErrInvalidChars
	}
	if len(id) == 0 {
		return ErrEmpty
	}
	if maxLen > 0 && len(id) > maxLen {
		return fmt.Errorf("(> %d chars): %w", maxLen, ErrTooLong)
	}
	return nil
}
