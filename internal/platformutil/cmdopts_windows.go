//go:build windows
// +build windows

package platformutil

import (
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

func SetPlatformOptions(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NO_WINDOW}
}
