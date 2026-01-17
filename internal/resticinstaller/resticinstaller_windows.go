package resticinstaller

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"

	"github.com/garethgeorge/backrest/internal/env"
	"go.uber.org/zap"
)

func findHelper() (string, error) {
	// Check if restic is provided via environment variable.
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

	// Windows specific logic: look for restic.exe adjacent to the executable.
	// We use os.Args[0] to determine the location of the running binary.
	resticInstallPath, _ := filepath.Abs(path.Join(path.Dir(os.Args[0]), "restic.exe"))
	if _, err := os.Stat(resticInstallPath); err == nil {
		// Found it. Check version but don't fail, just warn.
		if err := assertResticVersion(resticInstallPath, false /* strict */); err != nil {
			zap.S().Warnf("bundled restic binary %q may not be supported by backrest (this is expected if backrest was upgraded without the installer): %v", resticInstallPath, err)
		}
		return resticInstallPath, nil
	}

	// If not found on Windows, we fail. We do NOT auto-install/download.
	return "", fmt.Errorf("restic binary not found at %q (bundled installer missing?) and not in PATH", resticInstallPath)
}
