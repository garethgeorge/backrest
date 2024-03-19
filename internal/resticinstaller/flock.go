//go:build linux || darwin || freebsd
// +build linux darwin freebsd

package resticinstaller

import (
	"os"
	"path"
	"syscall"
)

func withFlock(lock string, do func() error) error {
	if err := os.MkdirAll(path.Dir(lock), 0700); err != nil {
		return err
	}
	f, err := os.Create(lock)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}

	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

	return do()
}
