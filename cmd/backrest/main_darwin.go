//go:build darwin

package main

import (
	"flag"
	"os"
	"strings"

	"go.uber.org/zap"
)

var darwinTray = flag.Bool("tray", false, "run with system tray applet (menu bar)")

func main() {
	flag.Parse()
	// Auto-enable tray mode when launched from a .app bundle
	inBundle := strings.Contains(os.Args[0], ".app/Contents/MacOS/")
	if *darwinTray || inBundle {
		startTray()
	} else {
		runApp()
	}
}

func reportError(err error) {
	zap.S().Errorf("backrest error: %v", err)
}
