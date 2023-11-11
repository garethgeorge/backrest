package restic

import (
	"context"
	"encoding/json"
	"testing"

	v1 "github.com/garethgeorge/resticui/gen/go/v1"
	test "github.com/garethgeorge/resticui/internal/test/helpers"
)

func TestResticInit(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()

	r := NewRepo(&v1.Repo{
		Id: "test",
		Uri: repo,
		Password: "test",
	}, WithRepoFlags("--no-cache"))

	r.init(context.Background())
}

func TestResticBackup(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()

	// create a new repo with cache disabled for testing
	r := NewRepo(&v1.Repo{
		Id: "test",
		Uri: repo,
		Password: "test",
	}, WithRepoFlags("--no-cache"))
	
	testData := test.CreateTestData(t)
	testData2 := test.CreateTestData(t)

	var tests = []struct {
		name string 
		opts []BackupOption
		files int // expected files at the end of the backup
		wantErr bool
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
		{
			name: "with nothing to backup",
			opts: []BackupOption{},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			summary, err := r.Backup(context.Background(), func(event *BackupEvent) {
				t.Logf("backup event: %v", event)
			}, tc.opts...)
			if (err != nil) != tc.wantErr {
				t.Fatalf("wanted error: %v, got: %v", tc.wantErr, err)
			}

			if tc.wantErr {
				return
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

func TestSnapshot(t *testing.T) {
	t.Parallel()

	repo := t.TempDir()

	r := NewRepo(&v1.Repo{
		Id: "test",
		Uri: repo,
		Password: "test",
	}, WithRepoFlags("--no-cache"))

	testData := test.CreateTestData(t)

	for i := 0; i < 10; i++ {
		_, err := r.Backup(context.Background(), nil, WithBackupPaths(testData))
		if err != nil {
			t.Fatalf("failed to backup and create new snapshot: %v", err)
		}
	}

	snapshots, err := r.Snapshots(context.Background())
	if err != nil {
		t.Fatalf("failed to list snapshots: %v", err)
	}

	if len(snapshots) != 10 {
		t.Errorf("wanted 10 snapshots, got: %d", len(snapshots))
	}

	data, _ := json.Marshal(snapshots)

	t.Logf("snapshots: %v", string(data))
}

func TestLs(t *testing.T) {
	t.Parallel()

	repo := t.TempDir()
	r := NewRepo(&v1.Repo{
		Id: "test",
		Uri: repo,
		Password: "test",
	}, WithRepoFlags("--no-cache"))

	testData := test.CreateTestData(t)

	snapshot, err := r.Backup(context.Background(), nil, WithBackupPaths(testData))
	if err != nil {
		t.Fatalf("failed to backup and create new snapshot: %v", err)
	}

	_, entries, err := r.ListDirectory(context.Background(), snapshot.SnapshotId, testData)

	if err != nil {
		t.Fatalf("failed to list directory: %v", err)
	}

	if len(entries) != 101 {
		t.Errorf("wanted 101 entries, got: %d", len(entries))
	}
}