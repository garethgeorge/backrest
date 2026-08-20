package repo

import (
	"os"
	"regexp"
	"strings"
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

// StripEnvValueQuotes removes a single matching pair of surrounding quotes
// (either " or ') from the VALUE portion of a KEY=VALUE string, if present.
//
// This exists because Go's os/exec sets process environment variables
// verbatim from the "KEY=VALUE" strings it is given -- unlike a shell, it
// does not strip quotes from the value. Users commonly write env vars in the
// KEY="VALUE" form out of habit (e.g. copying from a .env file or shell
// script), which would otherwise leave literal quote characters embedded in
// the value passed to restic, breaking things like base64-encoded secrets.
func StripEnvValueQuotes(kv string) string {
	idx := strings.Index(kv, "=")
	if idx == -1 {
		return kv
	}
	key := kv[:idx]
	value := kv[idx+1:]
	if len(value) >= 2 {
		first, last := value[0], value[len(value)-1]
		if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
			value = value[1 : len(value)-1]
		}
	}
	return key + "=" + value
}
