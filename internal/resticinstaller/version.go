package resticinstaller

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"

	"go.uber.org/zap"
)

func getResticVersion(binary string) (string, error) {
	cmd := exec.Command(binary, "version")
	out, err := cmd.Output()
	// check if error is a binary not found error
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return "", ErrResticNotFound
		}
		return "", fmt.Errorf("exec %v: %w", cmd.String(), err)
	}
	match := regexp.MustCompile(`restic\s+((\d+\.\d+\.\d+))`).FindSubmatch(out)
	if len(match) < 2 {
		return "", fmt.Errorf("could not find restic version in output: %s", out)
	}
	return string(match[1]), nil
}

func assertResticVersion(binary string, strict bool) error {
	if _, err := os.Stat(binary); err != nil {
		return fmt.Errorf("check if restic binary exists: %w", err)
	}

	if version, err := getResticVersion(binary); err != nil {
		return fmt.Errorf("determine restic version: %w", err)
	} else {
		cmp := compareSemVer(mustParseSemVer(version), requiredVersionSemver)
		if cmp < 0 {
			return fmt.Errorf("restic version %v is less than required version %v", version, RequiredResticVersion)
		} else if cmp > 0 && strict {
			return fmt.Errorf("restic version %v is newer than required version %v, it may not be supported by backrest", version, RequiredResticVersion)
		} else if cmp > 0 {
			zap.S().Warnf("restic version %v is newer than required version %v, it may not be supported by backrest", version, RequiredResticVersion)
		}
	}
	return nil
}

func parseSemVer(version string) ([3]int, error) {
	var major, minor, patch int
	_, err := fmt.Sscanf(version, "%d.%d.%d", &major, &minor, &patch)
	if err != nil {
		return [3]int{}, fmt.Errorf("invalid semantic version format: %w", err)
	}
	return [3]int{major, minor, patch}, nil
}

func mustParseSemVer(version string) [3]int {
	v, err := parseSemVer(version)
	if err != nil {
		panic(err)
	}
	return v
}

func compareSemVer(v1 [3]int, v2 [3]int) int {
	if v1[0] != v2[0] {
		return v1[0] - v2[0]
	}
	if v1[1] != v2[1] {
		return v1[1] - v2[1]
	}
	return v1[2] - v2[2]
}
