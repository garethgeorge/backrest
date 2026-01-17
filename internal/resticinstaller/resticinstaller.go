package resticinstaller

import (
	"errors"
	"sync"
)

var (
	ErrResticNotFound = errors.New("no restic binary")
)

var (
	RequiredResticVersion = "0.18.1"

	requiredVersionSemver = mustParseSemVer(RequiredResticVersion)

	tryFindRestic   sync.Once
	findResticErr   error
	foundResticPath string
)

// FindOrInstallResticBinary first tries to find the restic binary if provided as an environment variable. Otherwise it downloads restic if not already installed.
func FindOrInstallResticBinary() (string, error) {
	tryFindRestic.Do(func() {
		foundResticPath, findResticErr = findHelper()
	})

	if findResticErr != nil {
		return "", findResticErr
	}
	if foundResticPath == "" {
		return "", ErrResticNotFound
	}
	return foundResticPath, nil
}
