//go:build !windows
// +build !windows

package restic

func toPathFilter(path string) string {
	return path
}
