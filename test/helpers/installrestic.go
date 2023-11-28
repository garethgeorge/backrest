package helpers

import (
	"testing"

	"github.com/garethgeorge/resticui/internal/resticinstaller"
)

func ResticBinary(t *testing.T) string {
	binPath, err := resticinstaller.FindOrInstallResticBinary()
	if err != nil {
		t.Fatalf("find restic binary: %v", err)
	}
	return binPath
}

