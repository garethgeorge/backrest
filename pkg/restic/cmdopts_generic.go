//go:build !windows
// +build !windows

package restic

import (
	"os/exec"
)

func setPlatformOptions(cmd *exec.Cmd) {
	// No special options needed for non-Windows platforms
}
