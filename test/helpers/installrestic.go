package helpers

import (
	"testing"

	"github.com/garethgeorge/backrest/internal/resticinstaller"
)

func ResticBinary(t testing.TB) string {
	binPath, err := resticinstaller.FindOrInstallResticBinary()
	if err != nil {
		t.Fatalf("find restic binary: %v", err)
	}
	return binPath
}
