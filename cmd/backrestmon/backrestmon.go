//go:build windows || darwin
// +build windows darwin

package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/garethgeorge/backrest/internal/env"
	"github.com/getlantern/systray"
	"github.com/ncruces/zenity"

	_ "embed"
)

//go:embed icon.ico
var icon []byte

func main() {
	backrest, err := findBackrest()
	if err != nil {
		reportError(err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())

	cmd := exec.CommandContext(ctx, backrest)
	customizeCommand(cmd)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "ENV=production")

	if err := cmd.Start(); err != nil {
		reportError(err)
		cancel()
		return
	}

	systray.Run(func() {
		systray.SetTitle("Backrest Tray")
		systray.SetTooltip("Manage backrest")
		systray.SetIcon(icon)

		// First item: open the WebUI in the default browser.
		mOpenUI := systray.AddMenuItem("Open WebUI", "Open the Backrest WebUI in your default browser")
		mOpenUI.ClickedCh = make(chan struct{})
		go func() {
			for range mOpenUI.ClickedCh {
				// Parse address from env
				bindaddr := os.Getenv("BACKREST_PORT")
				if bindaddr == "" {
					bindaddr = ":9898"
				}

				// parse port from IP addr
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
			cancel()
			systray.Quit()
		}()
	}, func() {
		cancel()
	})

	if err := cmd.Wait(); err != nil {
		systray.Quit()
		if ctx.Err() != context.Canceled {
			reportError(fmt.Errorf("backrest process exited unexpectedly with error: %w", err))
		}
		return
	}
}

func findBackrest() (string, error) {
	// Backrest binary must be installed in the same directory as the backresttray binary.
	ex, err := os.Executable()
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(ex)

	wantPath := filepath.Join(dir, backrestBinName())

	if stat, err := os.Stat(wantPath); err == nil && !stat.IsDir() {
		return wantPath, nil
	}
	return "", fmt.Errorf("backrest binary not found at %s", wantPath)
}

func backrestBinName() string {
	if runtime.GOOS == "windows" {
		return "backrest.exe"
	} else {
		return "backrest"
	}
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
