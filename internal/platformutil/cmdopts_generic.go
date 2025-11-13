//go:build !windows
// +build !windows

package platformutil

import (
	"os/exec"
)

func SetPlatformOptions(cmd *exec.Cmd) {
	// No special options needed for non-Windows platforms
}
