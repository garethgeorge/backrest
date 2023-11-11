package restic

import (
	"context"
	"testing"

	v1 "github.com/garethgeorge/resticui/gen/go/v1"
	test "github.com/garethgeorge/resticui/internal/test/helpers"
)

func TestResticInit(t *testing.T) {
	repo := t.TempDir()

	r := NewRepo(&v1.Repo{
		Id: "test",
		Uri: repo,
		Password: "test",
	}, WithRepoFlags("--no-cache"))

	r.init(context.Background())
}

func TestResticBackup(t *testing.T) {
	repo := t.TempDir()

	// create a new repo with cache disabled for testing
	r := NewRepo(&v1.Repo{
		Id: "test",
		Uri: repo,
		Password: "test",
	}, WithRepoFlags("--no-cache"))

	r.init(context.Background())

	testData := test.CreateTestData(t)
	testData2 := test.CreateTestData(t)

	var tests = []struct {
		name string 
		opts []BackupOption
		files int // expected files at the end of the backup
	}{
		{
			name: "no options",
			opts: []BackupOption{WithBackupPaths(testData)},
			files: 100,
		},
		{
			name: "with two paths",
			opts:[]BackupOption{WithBackupPaths(testData), WithBackupPaths(testData2)},
			files: 200,
		},
		{
			name: "with exclude",
			opts: []BackupOption{WithBackupPaths(testData), WithBackupExcludes("file1*")},
			files: 90,
		},
		{
			name: "with exclude pattern",
			opts: []BackupOption{WithBackupPaths(testData), WithBackupExcludes("file*")},
			files: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			summary, err := r.Backup(context.Background(), func(event *BackupEvent) {
				t.Logf("backup event: %v", event)
			}, tc.opts...)
			if err != nil {
				t.Errorf("failed to backup: %v", err)
			}

			if summary == nil {
				t.Fatalf("wanted summary, got: nil")
			}

			if summary.TotalFilesProcessed != tc.files {
				t.Errorf("wanted %d files, got: %d", tc.files, summary.TotalFilesProcessed)
			}
		})
	}
}

