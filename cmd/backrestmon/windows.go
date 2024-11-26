//go:build windows
// +build windows

package main

import (
	"os/exec"
	"syscall"
)

func customizeCommand(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}
