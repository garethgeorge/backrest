package resticinstaller

import (
	"archive/zip"
	"bytes"
	"compress/bzip2"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/garethgeorge/backrest/internal/env"
	"github.com/gofrs/flock"
	"go.uber.org/zap"
)

var (
	ErrResticNotFound = errors.New("no restic binary")
)

var (
	RequiredResticVersion = "0.17.1"

	findResticMu  sync.Mutex
	didTryInstall bool
)

func resticBinName() string {
	if runtime.GOOS == "windows" {
		return fmt.Sprintf("restic-%v.exe", RequiredResticVersion)
	}
	return fmt.Sprintf("restic-%v", RequiredResticVersion)
}

func resticDownloadURL(version string) string {
	if runtime.GOOS == "windows" {
		// restic is only built for 386 and amd64 on Windows, default to amd64 for other platforms (e.g. arm64.)
		arch := "amd64"
		if runtime.GOARCH == "386" || runtime.GOARCH == "amd64" {
			arch = runtime.GOARCH
		}
		return fmt.Sprintf("https://github.com/restic/restic/releases/download/v%v/restic_%v_windows_%v.zip", version, version, arch)
	}
	return fmt.Sprintf("https://github.com/restic/restic/releases/download/v%v/restic_%v_%v_%v.bz2", version, version, runtime.GOOS, runtime.GOARCH)
}

func hashDownloadURL(version string) string {
	return fmt.Sprintf("https://github.com/restic/restic/releases/download/v%v/SHA256SUMS", version)
}

func sigDownloadURL(version string) string {
	return fmt.Sprintf("https://github.com/restic/restic/releases/download/v%v/SHA256SUMS.asc", version)
}

// getURL downloads the given url and returns the response body as a string.
func getURL(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("http GET %v: %w", url, err)
	}
	defer resp.Body.Close()

	var body bytes.Buffer
	_, err = io.Copy(&body, resp.Body)
	if err != nil {
		return nil, fmt.Errorf("copy response body to buffer: %w", err)
	}
	return body.Bytes(), nil
}

func verify(sha256 string) error {
	sha256sums, err := getURL(hashDownloadURL(RequiredResticVersion))
	if err != nil {
		return fmt.Errorf("get sha256sums: %w", err)
	}

	signature, err := getURL(sigDownloadURL(RequiredResticVersion))
	if err != nil {
		return fmt.Errorf("get signature: %w", err)
	}

	if ok, err := gpgVerify(sha256sums, signature); !ok || err != nil {
		return fmt.Errorf("gpg verification failed: ok=%v err=%v", ok, err)
	}

	if !strings.Contains(string(sha256sums), sha256) {
		fmt.Fprintf(os.Stderr, "sha256sums:\n%v\n", string(sha256sums))
		return fmt.Errorf("sha256sums do not contain %v", sha256)
	}

	return nil
}

// downloadFile downloads a file from the given url and saves it to the given path. The sha256 checksum of the file is returned on success.
func downloadFile(url string, downloadPath string) (string, error) {
	// Download ur as a file and save it to path
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/octet-stream")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("http GET %v: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("http GET %v: %v", url, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response body: %w", err)
	}
	hash := sha256.Sum256(body)

	if strings.HasSuffix(url, ".bz2") {
		zap.S().Infof("decompressing bz2 archive (size=%v)...", len(body))
		body, err = io.ReadAll(bzip2.NewReader(bytes.NewReader(body)))
		if err != nil {
			return "", fmt.Errorf("bz2 decompress body: %w", err)
		}
	} else if strings.HasSuffix(url, ".zip") {
		zap.S().Infof("decompressing zip archive (size=%v)...", len(body))

		archive, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
		if err != nil {
			return "", fmt.Errorf("open zip archive: %w", err)
		}

		if len(archive.File) != 1 {
			return "", fmt.Errorf("expected zip archive to contain exactly one file, got %v", len(archive.File))
		}
		f, err := archive.File[0].Open()
		if err != nil {
			return "", fmt.Errorf("open zip archive file %v: %w", archive.File[0].Name, err)
		}

		body, err = io.ReadAll(f)
		if err != nil {
			return "", fmt.Errorf("read zip archive file %v: %w", archive.File[0].Name, err)
		}
	}

	out, err := os.Create(downloadPath)
	if err != nil {
		return "", fmt.Errorf("create file %v: %w", downloadPath, err)
	}
	defer out.Close()
	if err != nil {
		return "", fmt.Errorf("create file %v: %w", downloadPath, err)
	}
	_, err = io.Copy(out, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("copy response body to file %v: %w", downloadPath, err)
	}

	return hex.EncodeToString(hash[:]), nil
}

func installResticIfNotExists(resticInstallPath string) error {
	lock := flock.New(filepath.Join(filepath.Dir(resticInstallPath), "install.lock"))
	if err := lock.Lock(); err != nil {
		return fmt.Errorf("lock %v: %w", lock.Path(), err)
	}
	defer lock.Unlock()

	if _, err := os.Stat(resticInstallPath); err == nil {
		// file is now installed, probably by another process. We can return.
		return nil
	}

	if err := os.MkdirAll(path.Dir(resticInstallPath), 0755); err != nil {
		return fmt.Errorf("create restic install directory %v: %w", path.Dir(resticInstallPath), err)
	}

	hash, err := downloadFile(resticDownloadURL(RequiredResticVersion), resticInstallPath+".tmp")
	if err != nil {
		return err
	}

	if err := verify(hash); err != nil {
		os.Remove(resticInstallPath) // try to remove the bad binary.
		return fmt.Errorf("failed to verify the authenticity of the downloaded restic binary: %v", err)
	}

	if err := os.Chmod(resticInstallPath+".tmp", 0755); err != nil {
		return fmt.Errorf("chmod executable %v: %w", resticInstallPath, err)
	}

	if err := os.Rename(resticInstallPath+".tmp", resticInstallPath); err != nil {
		return fmt.Errorf("rename %v.tmp to %v: %w", resticInstallPath, resticInstallPath, err)
	}

	return nil
}

func removeOldVersions(installDir string) {
	files, err := os.ReadDir(installDir)
	if err != nil {
		zap.S().Errorf("remove old restic versions: read dir %v: %v", installDir, err)
		return
	}

	for _, file := range files {
		if !strings.HasPrefix(file.Name(), "restic-") || strings.Contains(file.Name(), RequiredResticVersion) {
			continue
		}

		if err := os.Remove(path.Join(installDir, file.Name())); err != nil {
			zap.S().Errorf("remove old restic version %v: %v", file.Name(), err)
		}
	}
}

// FindOrInstallResticBinary first tries to find the restic binary if provided as an environment variable. Otherwise it downloads restic if not already installed.
func FindOrInstallResticBinary() (string, error) {
	findResticMu.Lock()
	defer findResticMu.Unlock()

	// Check if restic is provided.
	resticBin := env.ResticBinPath()
	if resticBin != "" {
		if _, err := os.Stat(resticBin); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return "", fmt.Errorf("stat(%v): %w", resticBin, err)
			}
			return "", fmt.Errorf("no binary found at path %v: %w", resticBin, ErrResticNotFound)
		}
		return resticBin, nil
	}

	// Search the PATH for the specific restic version.
	resticBinName := resticBinName()
	if binPath, err := exec.LookPath(resticBinName); err == nil {
		return binPath, nil
	}

	// Check for restic installation in data directory.
	resticInstallPath := path.Join(env.DataDir(), resticBinName)
	if runtime.GOOS == "windows" {
		// on windows use a path relative to the executable.
		resticInstallPath, _ = filepath.Abs(path.Join(path.Dir(os.Args[0]), resticBinName))
	}

	if err := os.MkdirAll(path.Dir(resticInstallPath), 0700); err != nil {
		return "", fmt.Errorf("create restic install directory %v: %w", path.Dir(resticInstallPath), err)
	}

	// Install restic if not found.
	if _, err := os.Stat(resticInstallPath); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("could not stat restic binary at %v: %w", resticBin, err)
		}

		if didTryInstall {
			return "", fmt.Errorf("already tried to install: %w", ErrResticNotFound)
		}
		didTryInstall = true

		zap.S().Infof("installing restic %v to %v...", resticInstallPath, RequiredResticVersion)
		if err := installResticIfNotExists(resticInstallPath); err != nil {
			return "", fmt.Errorf("install restic: %w", err)
		}
		zap.S().Infof("installed restic %v", RequiredResticVersion)
		removeOldVersions(path.Dir(resticInstallPath))
	}

	return resticInstallPath, nil
}
