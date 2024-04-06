//go:build windows
// +build windows

package restic

import (
	"path/filepath"
	"strings"
)

// toPathFilter converts a directory path to a path filter. Restic requires
// these to be absolute, starting with a forward slash. On Windows, we convert
// the drive letter to a single character and prepend a forward slash.
func toPathFilter(path string) string {
	cleanedPath := filepath.Clean(path)
	// If clean results in an empty string, ignoring the volume, it returns "."
	if cleanedPath != path+"." {
		path = cleanedPath
	}

	before, after, found := strings.Cut(path, string(filepath.Separator))
	if !found {
		// e.g. if path is "C:"
		before = path
		after = ""
	}

	if len(before) == 2 && before[1] == ':' && before[0] >= 'A' && before[0] <= 'Z' {
		path = filepath.Join(string(before[0]), after)
	}

	if path[0] != filepath.Separator {
		path = filepath.Join(string(filepath.Separator), path)
	}

	return filepath.ToSlash(path)
}
