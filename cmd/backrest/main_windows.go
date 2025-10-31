//go:build windows
// +build windows

package main

import (
	"flag"
	"fmt"
	"net"
	"os/exec"
	"runtime"

	"github.com/garethgeorge/backrest/internal/env"
	"github.com/getlantern/systray"
	"github.com/ncruces/zenity"

	_ "embed"
)

//go:embed icon.ico
var icon []byte

var windowsTray = flag.Bool("windows-tray", true, "run the windows tray application")

func main() {
	flag.Parse()
	if *windowsTray {
		startTray()
	} else {
		runApp()
	}
}

func startTray() {
	go runApp()

	systray.Run(func() {
		systray.SetTitle("Backrest Tray")
		systray.SetTooltip("Manage backrest")
		systray.SetIcon(icon)

		// First item: open the WebUI in the default browser.
		mOpenUI := systray.AddMenuItem("Open WebUI", "Open the Backrest WebUI in your default browser")
		mOpenUI.ClickedCh = make(chan struct{})
		go func() {
			for range mOpenUI.ClickedCh {
				bindaddr := env.BindAddress()
				if bindaddr == "" {
					bindaddr = ":9898"
				}
				_, port, err := net.SplitHostPort(bindaddr)
				if err != nil {
					port = "9898" // try the default
				}
				if err := openBrowser(fmt.Sprintf("http://localhost:%v", port)); err != nil {
					reportError(err)
				}
			}
		}()

		// Second item: open the log file in the file explorer
		mOpenLog := systray.AddMenuItem("Open Log Dir", "Open the Backrest log directory")
		mOpenLog.ClickedCh = make(chan struct{})
		go func() {
			for range mOpenLog.ClickedCh {
				cmd := exec.Command(`explorer`, `/select,`, env.LogsPath())
				cmd.Start()
				go cmd.Wait()
			}
		}()

		// Last item: quit button to stop the backrest process.
		mQuit := systray.AddMenuItem("Quit", "Kills the backrest process and exits the tray app")
		mQuit.ClickedCh = make(chan struct{})
		go func() {
			<-mQuit.ClickedCh
			systray.Quit()
		}()
	}, func() {
	})
}

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "linux":
		return exec.Command("xdg-open", url).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return fmt.Errorf("unsupported platform")
	}
}

func reportError(err error) {
	zenity.Error(err.Error(), zenity.Title("Backrest Error"))
}
