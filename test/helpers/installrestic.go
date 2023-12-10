package helpers

import (
	"testing"

	"github.com/garethgeorge/restora/internal/resticinstaller"
)

func ResticBinary(t *testing.T) string {
	binPath, err := resticinstaller.FindOrInstallResticBinary()
	if err != nil {
		t.Fatalf("find restic binary: %v", err)
	}
	return binPath
}
