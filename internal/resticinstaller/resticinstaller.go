package resticinstaller

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
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
	RequiredResticVersion = "0.17.3"

	findResticMu  sync.Mutex
	didTryInstall bool
)

func getResticVersion(binary string) (string, error) {
	cmd := exec.Command(binary, "version")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("exec %v: %w", cmd.String(), err)
	}
	match := regexp.MustCompile(`restic\s+((\d+\.\d+\.\d+))`).FindSubmatch(out)
	if len(match) < 2 {
		return "", fmt.Errorf("could not find restic version in output: %s", out)
	}
	return string(match[1]), nil
}

func assertResticVersion(binary string) error {
	if version, err := getResticVersion(binary); err != nil {
		return fmt.Errorf("determine restic version: %w", err)
	} else if version != RequiredResticVersion {
		return fmt.Errorf("want restic %v but found version %v", RequiredResticVersion, version)
	}
	return nil
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

func installResticIfNotExists(resticInstallPath string) error {
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
		if err := assertResticVersion(resticBin); err != nil {
			zap.S().Warnf("restic binary %q may not be supported by backrest", resticBin, err)
		}

		if _, err := os.Stat(resticBin); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return "", fmt.Errorf("stat(%v): %w", resticBin, err)
			}
			return "", fmt.Errorf("no binary found at path %v: %w", resticBin, ErrResticNotFound)
		}
		return resticBin, nil
	}

	// Search the PATH for the specific restic version.
	if binPath, err := exec.LookPath("restic"); err == nil {
		if err := assertResticVersion(binPath); err == nil {
			zap.S().Infof("restic binary %q in $PATH matches required version %v, it will be used for backrest commands", binPath, RequiredResticVersion)
			return binPath, nil
		} else {
			zap.S().Infof("restic binary %q in $PATH is not being used, it may not be supported by backrest: %v", binPath, err)
		}
	}

	// Check for restic installation in data directory.
	var resticInstallPath string
	if runtime.GOOS == "windows" {
		// on windows use a path relative to the executable.
		resticInstallPath, _ = filepath.Abs(path.Join(path.Dir(os.Args[0]), "restic"))
	} else {
		resticInstallPath = filepath.Join(env.DataDir(), "restic")
	}
	if err := os.MkdirAll(filepath.Dir(env.DataDir()), 0700); err != nil {
		return "", fmt.Errorf("create restic install directory %v: %w", path.Dir(resticInstallPath), err)
	}

	// Install restic if not found OR if the version is not the required version
	if err := assertResticVersion(resticInstallPath); err != nil {
		lock := flock.New(filepath.Join(filepath.Dir(resticInstallPath), "install.lock"))
		if err := lock.Lock(); err != nil {
			return "", fmt.Errorf("lock %v: %w", lock.Path(), err)
		}
		defer lock.Unlock()

		if _, err := os.Stat(resticInstallPath); err == nil {
			zap.S().Infof("reinstalling restic binary in data dir due to failed checks: %w", err)
			if err := os.Remove(resticInstallPath); err != nil {
				return "", fmt.Errorf("remove old restic binary %v: %w", resticInstallPath, err)
			}
		}

		if didTryInstall {
			return "", fmt.Errorf("already tried to install: %w", ErrResticNotFound)
		}
		didTryInstall = true

		zap.S().Infof("downloading restic %v to %v...", RequiredResticVersion, resticInstallPath)
		if err := installResticIfNotExists(resticInstallPath); err != nil {
			return "", fmt.Errorf("install restic: %w", err)
		}
		zap.S().Infof("installed restic %v", RequiredResticVersion)

		// TODO: this check is no longer needed, remove it after a few releases.
		removeOldVersions(path.Dir(resticInstallPath))
	}

	zap.S().Infof("restic binary %v in data dir will be used as no system install matching required version %v is found", resticInstallPath, RequiredResticVersion)
	return resticInstallPath, nil
}
