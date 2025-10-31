//go:build windows
// +build windows

package restic

import (
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

func setPlatformOptions(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NO_WINDOW}
}
