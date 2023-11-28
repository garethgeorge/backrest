package resticinstaller

import (
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
	return fmt.Sprintf("https://github.com/restic/restic/releases/download/v%v/restic_%v_%v_%v.bz2", version, version, runtime.GOOS, runtime.GOARCH)
}

func downloadFile(url string, path string) error {
	// Download ur as a file and save it to path
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("http GET %v: %w", url, err)
	}
	defer resp.Body.Close()

	var body io.Reader = resp.Body
	if strings.HasSuffix(url, ".bz2") {
		body = bzip2.NewReader(resp.Body)
	}

	out, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create file %v: %w", path, err)
	}
	defer out.Close()
	if err != nil {
		return fmt.Errorf("create file %v: %w", path, err)
	}
	_, err = io.Copy(out, body)
	if err != nil {
		return fmt.Errorf("copy response body to file %v: %w", path, err)
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
	if _, err := os.Stat(resticInstallPath); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("could not stat restic binary at %v: %w", resticBin, err)
		}

		if didTryInstall {
			return "", fmt.Errorf("already tried to install: %w", ErrResticNotFound)
		}
		didTryInstall = true

		if err := installResticIfNotExists(resticInstallPath); err != nil {
			return "", fmt.Errorf("install restic: %w", err)
		}
	}

	return resticInstallPath, nil
}
