//go:build !tray

package main

import "fmt"

func startTray() {
	fmt.Println("error: this binary was built without system tray support.")
	fmt.Println("Use the tray-enabled build, or run without the --tray flag.")
	runApp()
}
