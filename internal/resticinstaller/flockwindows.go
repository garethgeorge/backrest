//go:build windows
// +build windows

package resticinstaller

func withFlock(lock string, do func() error) error {
	// TODO: windows file locking. Not a major issue as locking is only needed for test runs.
	return do()
}
