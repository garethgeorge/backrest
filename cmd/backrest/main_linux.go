//go:build linux

package main

import (
	"flag"

	"go.uber.org/zap"
)

var linuxTray = flag.Bool("tray", false, "run with system tray applet")

func main() {
	flag.Parse()
	if *linuxTray {
		startTray()
	} else {
		runApp()
	}
}

func reportError(err error) {
	zap.S().Errorf("backrest error: %v", err)
}
