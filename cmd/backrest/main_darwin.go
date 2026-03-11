//go:build darwin

package main

import (
	"flag"

	"go.uber.org/zap"
)

var darwinTray = flag.Bool("tray", false, "run with system tray applet (menu bar)")

func main() {
	flag.Parse()
	if *darwinTray {
		startTray()
	} else {
		runApp()
	}
}

func reportError(err error) {
	zap.S().Errorf("backrest error: %v", err)
}
