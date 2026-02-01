//go:build windows
// +build windows

package platformutil

import (
	"os/exec"
	"syscall"
)

func SetPlatformOptions(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.HideWindow = true
}
