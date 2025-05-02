package resticinstaller

import (
	"fmt"
	"runtime"
)

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
