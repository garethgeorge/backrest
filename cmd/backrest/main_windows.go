//go:build windows

package main

import (
	"flag"

	"github.com/ncruces/zenity"
)

var windowsTray = flag.Bool("windows-tray", true, "run the windows tray application")

func main() {
	flag.Parse()
	if *windowsTray {
		startTray()
	} else {
		runApp()
	}
}

func reportError(err error) {
	zenity.Error(err.Error(), zenity.Title("Backrest Error"))
}
