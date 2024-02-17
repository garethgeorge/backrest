package helpers

import (
	"fmt"
	"os"
	"path"
	"testing"
)

func CreateTestData(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	for i := 0; i < 100; i++ {
		err := os.WriteFile(path.Join(dir, fmt.Sprintf("file%2d", i)), []byte(fmt.Sprintf("test data %d", i)), 0644)
		if err != nil {
			t.Fatalf("failed to create test data: %v", err)
		}
	}
	return dir
}

func CreateUnreadable(t *testing.T, path string) {
	t.Helper()

	// Create a file that can be written but can't be read by the current user
	err := os.WriteFile(path, []byte("test data"), 0200)
	if err != nil {
		t.Fatalf("failed to create unreadable file: %v", err)
	}
}
