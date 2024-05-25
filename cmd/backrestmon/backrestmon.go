//go:build windows
// +build windows

package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/garethgeorge/backrest/internal/env"
	"github.com/getlantern/systray"
	"github.com/ncruces/zenity"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"

	_ "embed"
)

//go:embed icon.ico
var icon []byte

func main() {
	l, err := createLogWriter()
	if err != nil {
		reportError(err)
		return
	}
	defer l.Close()

	backrest, err := findBackrest()
	if err != nil {
		reportError(err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())

	cmd := exec.CommandContext(ctx, backrest)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "ENV=production")

	pro, pwo := io.Pipe()
	pre, pwe := io.Pipe()
	cmd.Stdout = pwo
	cmd.Stderr = pwe

	go func() {
		io.Copy(l, io.MultiReader(pro, pre))
	}()

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
				if err := openBrowser("http://localhost:9898"); err != nil {
					reportError(err)
				}
			}
		}()

		// Second item: open the log file in the file explorer
		mOpenLog := systray.AddMenuItem("Open Log Dir", "Open the Backrest log directory")
		mOpenLog.ClickedCh = make(chan struct{})
		go func() {
			for range mOpenLog.ClickedCh {
				cmd := exec.Command(`explorer`, `/select,`, logsPath())
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

func createLogWriter() (io.WriteCloser, error) {
	logsDir := logsPath()
	fmt.Printf("Logging to %s\n", logsDir)
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return nil, err
	}

	l := &lumberjack.Logger{
		Filename:   filepath.Join(logsDir, "backrest.log"),
		MaxSize:    5, // megabytes
		MaxBackups: 3,
		MaxAge:     14,
		Compress:   true,
	}

	return l, nil
}

func logsPath() string {
	dataDir := env.DataDir()
	return filepath.Join(dataDir, "processlogs")
}
