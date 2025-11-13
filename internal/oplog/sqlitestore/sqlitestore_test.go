package sqlitestore

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestBackup_NewDatabaseCreatesValidBackup verifies that a new database creates
// a backup during initialization and that the backup is valid and can be read.
func TestBackup_NewDatabaseCreatesValidBackup(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	store, err := NewSqliteStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// NewSqliteStore should have created an initial backup
	matches, err := filepath.Glob(filepath.Join(tempDir, "test.db-*.backup"))
	if err != nil {
		t.Fatalf("failed to glob for backups: %v", err)
	}

	if len(matches) != 1 {
		t.Fatalf("expected 1 backup file from NewSqliteStore, got %d", len(matches))
	}

	// Verify the backup file exists and is not empty
	info, err := os.Stat(matches[0])
	if err != nil {
		t.Fatalf("failed to stat backup file: %v", err)
	}

	if info.Size() == 0 {
		t.Fatal("backup file is empty")
	}

	// Try to open the backup as a SQLite database to verify it's valid
	backupStore, err := NewSqliteStore(matches[0])
	if err != nil {
		t.Fatalf("backup file is not a valid SQLite database: %v", err)
	}
	defer backupStore.Close()
}

// TestBackup_CleansUpOldBackups verifies that after running backup multiple times,
// it correctly cleans up old backups and keeps only the specified number.
func TestBackup_CleansUpOldBackups(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	store, err := NewSqliteStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Clear the initial backup created during NewSqliteStore
	matches, _ := filepath.Glob(filepath.Join(tempDir, "test.db-*.bak"))
	for _, match := range matches {
		os.Remove(match)
	}

	// Create 10 backups with force=true, keeping only 3
	for i := 0; i < 10; i++ {
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
		err = store.backup(dbPath, 3, true)
		if err != nil {
			t.Fatalf("failed to run backup %d: %v", i, err)
		}
	}

	// Should have only 3 backups remaining
	matches, err = filepath.Glob(filepath.Join(tempDir, "test.db-*.backup"))
	if err != nil {
		t.Fatalf("failed to glob for backups: %v", err)
	}

	if len(matches) != 3 {
		t.Errorf("expected 3 backups (keepCount=3), got %d", len(matches))
	}
}
