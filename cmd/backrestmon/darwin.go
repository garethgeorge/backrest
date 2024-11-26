//go:build darwin
// +build darwin

package main

import (
	"os/exec"
)

func customizeCommand(cmd *exec.Cmd) {
	// No customization needed on macOS
}
