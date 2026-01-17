//go:build !windows

package resticinstaller

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/garethgeorge/backrest/internal/env"
	"github.com/gofrs/flock"
	"go.uber.org/zap"
)

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

func installRestic(targetPath string) error {
	sha256sum, err := downloadFile(resticDownloadURL(RequiredResticVersion), targetPath+".tmp")
	if err != nil {
		return fmt.Errorf("downloading: %w", err)
	}

	if err := verify(sha256sum); err != nil {
		return fmt.Errorf("verifying: %w", err)
	}

	if err := os.Rename(targetPath+".tmp", targetPath); err != nil {
		return fmt.Errorf("renaming %v: %w", targetPath, err)
	}

	if err := os.Chmod(targetPath, 0755); err != nil {
		return fmt.Errorf("chmod executable %v: %w", targetPath, err)
	}

	return nil
}

func findOrDownloadRestic(installPath string) error {
	if err := assertResticVersion(installPath, true /* strict */); err == nil {
		return nil
	}

	lock := flock.New(filepath.Join(filepath.Dir(installPath), "install.lock"))
	if err := lock.Lock(); err != nil {
		return fmt.Errorf("acquire lock on restic install dir %v: %v", lock.Path(), err)
	}
	defer lock.Unlock()

	if err := assertResticVersion(installPath, true /* strict */); err == nil {
		return nil
	} else {
		zap.S().Infof("restic binary %v failed version validation: %v", installPath, err)
	}

	zap.S().Infof("installing restic to %v", installPath)
	if err := installRestic(installPath); err != nil {
		return fmt.Errorf("install restic: %w", err)
	}

	return nil
}

func findHelper() (string, error) {
	// Check if restic is provided.
	resticBinOverride := env.ResticBinPath()
	if resticBinOverride != "" {
		if err := assertResticVersion(resticBinOverride, false /* strict */); err != nil {
			zap.S().Warnf("restic binary %q may not be supported by backrest: %v", resticBinOverride, err)
		}

		if _, err := os.Stat(resticBinOverride); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return "", fmt.Errorf("check if restic binary exists at %v: %v", resticBinOverride, err)
			}
			return "", fmt.Errorf("no restic binary found at %v", resticBinOverride)
		}
		return resticBinOverride, nil
	}

	// Search the PATH for the specific restic version.
	if binPath, err := exec.LookPath("restic"); err == nil {
		if err := assertResticVersion(binPath, false /* strict */); err == nil {
			zap.S().Infof("restic binary %q in $PATH matches required version %v, it will be used for backrest commands", binPath, RequiredResticVersion)
			return binPath, nil
		} else {
			zap.S().Infof("restic binary %q in $PATH is not being used, it may not be supported by backrest: %v", binPath, err)
		}
	}

	// Check for restic installation in data directory.
	resticInstallPath := filepath.Join(env.DataDir(), "restic")

	if err := os.MkdirAll(filepath.Dir(resticInstallPath), 0700); err != nil {
		return "", fmt.Errorf("create restic install directory %v: %w", path.Dir(resticInstallPath), err)
	}

	if err := findOrDownloadRestic(resticInstallPath); err != nil {
		return "", fmt.Errorf("find or download restic: %w", err)
	}

	zap.S().Infof("restic binary %q in data dir matches required version %v, it will be used for backrest commands", resticInstallPath, RequiredResticVersion)
	return resticInstallPath, nil
}
