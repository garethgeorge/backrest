package repo

import (
	"os"
	"regexp"
)

var (
	envVarSubstRegex = regexp.MustCompile(`\${[^}]*}`)
)

// ExpandEnv expands environment variables of the form ${VAR} in a string.
func ExpandEnv(s string) string {
	return envVarSubstRegex.ReplaceAllStringFunc(s, func(match string) string {
		e, _ := os.LookupEnv(match[2 : len(match)-1])
		return e
	})
}
