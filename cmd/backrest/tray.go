//go:build tray

package main

import (
	"fmt"
	"net"
	"os/exec"
	"runtime"

	"fyne.io/systray"
	"github.com/garethgeorge/backrest/internal/env"
)

func startTray() {
	status := newTrayStatus()
	// Observe oplog status in-process so the icon can reflect backup state.
	onOpLogReady = status.attach
	go func() {
		runApp()
		systray.Quit()
	}()
	systray.Run(func() { onReady(status) }, func() {})
}

var mStatusText *systray.MenuItem

func onReady(status *trayStatus) {
	systray.SetTooltip("Backrest")
	systray.SetIcon(icon)

	if runtime.GOOS == "darwin" {
		mStatusText = systray.AddMenuItem("", "")
		mStatusText.Disable()
		mStatusText.Hide()
	}

	mOpenUI := systray.AddMenuItem("Open WebUI", "Open the Backrest WebUI in your default browser")
	mOpenLog := systray.AddMenuItem("Open Log Dir", "Open the Backrest log directory")
	mQuit := systray.AddMenuItem("Quit", "Kills the backrest process and exits the tray app")

	// The tray is live, so icon writes now take effect: start reflecting status.
	go status.run()

	go func() {
		for {
			select {
			case <-mOpenUI.ClickedCh:
				bindaddr := env.BindAddress()
				if bindaddr == "" {
					bindaddr = ":9898"
				}
				_, port, err := net.SplitHostPort(bindaddr)
				if err != nil {
					port = "9898"
				}
				if err := openBrowser(fmt.Sprintf("http://localhost:%v", port)); err != nil {
					reportError(err)
				}
			case <-mOpenLog.ClickedCh:
				openFileManager(env.LogsPath())
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
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

func openFileManager(dir string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", `/select,`, dir)
	case "darwin":
		cmd = exec.Command("open", dir)
	case "linux":
		cmd = exec.Command("xdg-open", dir)
	default:
		return
	}
	cmd.Start()
	go cmd.Wait()
}
