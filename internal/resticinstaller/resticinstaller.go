package resticinstaller

import (
	"archive/zip"
	"bytes"
	"compress/bzip2"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"runtime"
	"strings"
	"sync"

	"github.com/garethgeorge/resticui/internal/config"
	"go.uber.org/zap"
)

var (
	ErrResticNotFound = errors.New("no restic binary")
)

var (
	RequiredResticVersion = "0.16.2"

	findResticMu  sync.Mutex
	didTryInstall bool
)

func resticDownloadURL(version string) string {
	if runtime.GOOS == "windows" {
		return fmt.Sprintf("https://github.com/restic/restic/releases/download/v%v/restic_%v_windows_%v.zip", version, version, runtime.GOARCH)
	}
	return fmt.Sprintf("https://github.com/restic/restic/releases/download/v%v/restic_%v_%v_%v.bz2", version, version, runtime.GOOS, runtime.GOARCH)
}

func downloadFile(url string, downloadPath string) error {
	// Download ur as a file and save it to path
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("http GET %v: %w", url, err)
	}
	defer resp.Body.Close()

	var body io.Reader = resp.Body
	if strings.HasSuffix(url, ".bz2") {
		body = bzip2.NewReader(resp.Body)
	} else if strings.HasSuffix(url, ".zip") {
		var fullBody bytes.Buffer
		_, err := io.Copy(&fullBody, resp.Body)
		if err != nil {
			return fmt.Errorf("copy response body to buffer: %w", err)
		}

		archive, err := zip.NewReader(bytes.NewReader(fullBody.Bytes()), int64(fullBody.Len()))
		if err != nil {
			return fmt.Errorf("open zip archive: %w", err)
		}

		if len(archive.File) != 1 {
			return fmt.Errorf("expected zip archive to contain exactly one file, got %v", len(archive.File))
		}
		body, err = archive.File[0].Open()
		if err != nil {
			return fmt.Errorf("open zip archive file %v: %w", archive.File[0].Name, err)
		}
	}

	out, err := os.Create(downloadPath)
	if err != nil {
		return fmt.Errorf("create file %v: %w", downloadPath, err)
	}
	defer out.Close()
	if err != nil {
		return fmt.Errorf("create file %v: %w", downloadPath, err)
	}
	_, err = io.Copy(out, body)
	if err != nil {
		return fmt.Errorf("copy response body to file %v: %w", downloadPath, err)
	}

	return nil
}

func downloadExecutable(url string, path string) error {
	if err := downloadFile(url, path+".tmp"); err != nil {
		return err
	}

	if err := os.Chmod(path+".tmp", 0755); err != nil {
		return fmt.Errorf("chmod executable %v: %w", path, err)
	}

	if err := os.Rename(path+".tmp", path); err != nil {
		return fmt.Errorf("rename %v.tmp to %v: %w", path, path, err)
	}

	return nil
}

func installResticIfNotExists(resticInstallPath string) error {
	// withFlock is used to ensure tests pass; when running on CI multiple tests may try to install restic at the same time.
	return withFlock(path.Join(config.DataDir(), "install.lock"), func() error {
		if _, err := os.Stat(resticInstallPath); err == nil {
			// file is now installed, probably by another process. We can return.
			return nil
		}

		if err := os.MkdirAll(path.Dir(resticInstallPath), 0755); err != nil {
			return fmt.Errorf("create restic install directory %v: %w", path.Dir(resticInstallPath), err)
		}

		if err := downloadExecutable(resticDownloadURL(RequiredResticVersion), resticInstallPath); err != nil {
			return fmt.Errorf("download restic version %v: %w", RequiredResticVersion, err)
		}

		return nil
	})
}

// FindOrInstallResticBinary first tries to find the restic binary if provided as an environment variable. Otherwise it downloads restic if not already installed.
func FindOrInstallResticBinary() (string, error) {
	findResticMu.Lock()
	defer findResticMu.Unlock()

	// Check if restic is provided.
	resticBin := config.ResticBinPath()
	if resticBin != "" {
		if _, err := os.Stat(resticBin); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return "", fmt.Errorf("stat(%v): %w", resticBin, err)
			}
			return "", fmt.Errorf("no binary found at path %v: %w", resticBin, ErrResticNotFound)
		}
		return resticBin, nil
	}

	// Check for restic installation in data directory.
	resticInstallPath := path.Join(config.DataDir(), fmt.Sprintf("restic-%v", RequiredResticVersion))
	if runtime.GOOS == "windows" {
		programFiles := os.Getenv("programfiles(x86)")
		if programFiles == "" {
			programFiles = os.Getenv("programfiles")
		}
		resticInstallPath = path.Join(programFiles, "restic", fmt.Sprintf("restic-%v.exe", RequiredResticVersion))
	}
	if _, err := os.Stat(resticInstallPath); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("could not stat restic binary at %v: %w", resticBin, err)
		}

		if didTryInstall {
			return "", fmt.Errorf("already tried to install: %w", ErrResticNotFound)
		}
		didTryInstall = true

		zap.S().Infof("Installing restic %v to %v...", resticInstallPath, RequiredResticVersion)
		if err := installResticIfNotExists(resticInstallPath); err != nil {
			return "", fmt.Errorf("install restic: %w", err)
		}
		zap.S().Infof("Installed restic %v", RequiredResticVersion)
	}

	return resticInstallPath, nil
}
