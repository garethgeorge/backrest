package restic

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/garethgeorge/backrest/test/helpers"
)

func TestResticInit(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()

	r := NewRepo(helpers.ResticBinary(t), repo, WithFlags("--no-cache"), WithEnv("RESTIC_PASSWORD=test"))

	if err := r.Init(context.Background()); err != nil {
		t.Fatalf("failed to init repo: %v", err)
	}
}

func TestResticBackup(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()

	// create a new repo with cache disabled for testing
	r := NewRepo(helpers.ResticBinary(t), repo, WithFlags("--no-cache"), WithEnv("RESTIC_PASSWORD=test"))
	if err := r.Init(context.Background()); err != nil {
		t.Fatalf("failed to init repo: %v", err)
	}

	testData := helpers.CreateTestData(t)
	testData2 := helpers.CreateTestData(t)
	testDataUnreadable := t.TempDir()
	helpers.CreateUnreadable(t, testDataUnreadable+"/unreadable")

	var tests = []struct {
		name     string
		opts     []GenericOption
		paths    []string
		files    int64 // expected files at the end of the backup
		wantErr  bool
		unixOnly bool
	}{
		{
			name:  "no options",
			paths: []string{testData},
			opts:  []GenericOption{},
			files: 100,
		},
		{
			name:  "with two paths",
			paths: []string{testData, testData2},
			opts:  []GenericOption{},
			files: 200,
		},
		{
			name:  "with exclude",
			paths: []string{testData},
			opts:  []GenericOption{WithFlags("--exclude", "file1*")},
			files: 90,
		},
		{
			name:  "with exclude pattern",
			paths: []string{testData},
			opts:  []GenericOption{WithFlags("--iexclude=file*")},
			files: 0,
		},
		{
			name:    "with nothing to backup",
			paths:   []string{},
			opts:    []GenericOption{},
			wantErr: true,
		},
		{
			name:    "with unreadable file",
			paths:   []string{testData, testDataUnreadable},
			opts:    []GenericOption{},
			wantErr: true,
		},
		{
			name:  "with wrapper process",
			paths: []string{testData},
			opts: []GenericOption{
				WithPrefixCommand("nice", "-n", "19"),
			},
			files:    100,
			unixOnly: true,
		},
		{
			name:  "with invalid wrapper process",
			paths: []string{testData},
			opts: []GenericOption{
				WithPrefixCommand("invalid-wrapper"),
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if runtime.GOOS == "windows" && tc.unixOnly {
				t.Skip("test is unix only")
			}

			gotEvent := false
			summary, err := r.Backup(context.Background(), tc.paths, func(event *BackupProgressEntry) {
				t.Logf("backup event: %v", event)
				gotEvent = true
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

			if !gotEvent {
				t.Errorf("wanted backup event, got: false")
			}
		})
	}
}

func TestResticPartialBackup(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()

	// create a new repo with cache disabled for testing
	r := NewRepo(helpers.ResticBinary(t), repo, WithFlags("--no-cache"), WithEnv("RESTIC_PASSWORD=test"))
	if err := r.Init(context.Background()); err != nil {
		t.Fatalf("failed to init repo: %v", err)
	}

	testDataUnreadable := t.TempDir()
	unreadablePath := filepath.Join(testDataUnreadable, "unreadable")
	helpers.CreateUnreadable(t, unreadablePath)

	var entries []BackupProgressEntry

	summary, err := r.Backup(context.Background(), []string{testDataUnreadable}, func(entry *BackupProgressEntry) {
		entries = append(entries, *entry)
	})
	if !errors.Is(err, ErrPartialBackup) {
		t.Fatalf("wanted error to be partial backup, got: %v", err)
	}
	if summary == nil {
		t.Fatalf("wanted summary, got: nil")
	}

	if !slices.ContainsFunc(entries, func(e BackupProgressEntry) bool {
		return e.MessageType == "error" && strings.Contains(e.Item, unreadablePath)
	}) {
		t.Errorf("wanted entries to contain an error event for the unreadable file (%s), but did not find it", unreadablePath)
		t.Logf("entries:\n")
		for _, entry := range entries {
			t.Logf("%+v\n", entry)
		}
	}
}

func TestResticBackupLots(t *testing.T) {
	t.Parallel()
	t.Skip("this test takes a long time to run")

	repo := t.TempDir()

	// create a new repo with cache disabled for testing
	r := NewRepo(helpers.ResticBinary(t), repo, WithFlags("--no-cache"), WithEnv("RESTIC_PASSWORD=test"))
	if err := r.Init(context.Background()); err != nil {
		t.Fatalf("failed to init repo: %v", err)
	}

	testData := helpers.CreateTestData(t)

	// backup 25 times
	for i := 0; i < 25; i++ {
		_, err := r.Backup(context.Background(), []string{testData}, func(e *BackupProgressEntry) {
			t.Logf("backup event: %+v", e)
		})
		if err != nil {
			t.Fatalf("failed to backup and create new snapshot: %v", err)
		}
	}
}

func TestSnapshot(t *testing.T) {
	t.Parallel()

	repo := t.TempDir()

	r := NewRepo(helpers.ResticBinary(t), repo, WithFlags("--no-cache"), WithEnv("RESTIC_PASSWORD=test"))
	if err := r.Init(context.Background()); err != nil {
		t.Fatalf("failed to init repo: %v", err)
	}

	testData := helpers.CreateTestData(t)

	for i := 0; i < 10; i++ {
		_, err := r.Backup(context.Background(), []string{testData}, nil, WithFlags("--tag", fmt.Sprintf("tag%d", i)))
		if err != nil {
			t.Fatalf("failed to backup and create new snapshot: %v", err)
		}
	}

	var tests = []struct {
		name                string
		opts                []GenericOption
		count               int
		checkSnapshotFields bool
	}{
		{
			name:  "no options",
			opts:  []GenericOption{},
			count: 10,
		},
		{
			name:                "with tag",
			opts:                []GenericOption{WithTags("tag1")},
			count:               1,
			checkSnapshotFields: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			snapshots, err := r.Snapshots(context.Background(), tc.opts...)
			if err != nil {
				t.Fatalf("failed to list snapshots: %v", err)
			}

			if len(snapshots) != tc.count {
				t.Errorf("wanted %d snapshots, got: %d", tc.count, len(snapshots))
			}

			// Ensure that snapshot timestamps are set, this is critical for correct ordering in the orchestrator.
			for _, snapshot := range snapshots {
				if snapshot.UnixTimeMs() == 0 {
					t.Errorf("wanted snapshot time to be non-zero, got: %v", snapshot.UnixTimeMs())
				}
				if snapshot.SnapshotSummary.DurationMs() == 0 {
					t.Errorf("wanted snapshot duration to be non-zero, got: %v", snapshot.SnapshotSummary.DurationMs())
				}
				if tc.checkSnapshotFields {
					checkSnapshotFieldsHelper(t, snapshot)
				}
			}
		})
	}
}

func checkSnapshotFieldsHelper(t *testing.T, snapshot *Snapshot) {
	if snapshot.Id == "" {
		t.Errorf("wanted snapshot ID to be non-empty, got: %v", snapshot.Id)
	}
	if snapshot.Tree == "" {
		t.Errorf("wanted snapshot tree to be non-empty, got: %v", snapshot.Tree)
	}
	if snapshot.Hostname == "" {
		t.Errorf("wanted snapshot hostname to be non-empty, got: %v", snapshot.Hostname)
	}
	if snapshot.Username == "" {
		t.Errorf("wanted snapshot username to be non-empty, got: %v", snapshot.Username)
	}
	if len(snapshot.Paths) == 0 {
		t.Errorf("wanted snapshot paths to be non-empty, got: %v", snapshot.Paths)
	}
	if len(snapshot.Tags) == 0 {
		t.Errorf("wanted snapshot tags to be non-empty, got: %v", snapshot.Tags)
	}
	if snapshot.UnixTimeMs() == 0 {
		t.Errorf("wanted snapshot time to be non-zero, got: %v", snapshot.UnixTimeMs())
	}
	if snapshot.SnapshotSummary.TreeBlobs == 0 {
		t.Errorf("wanted snapshot tree blobs to be non-zero, got: %v", snapshot.SnapshotSummary.TreeBlobs)
	}
	if snapshot.SnapshotSummary.DataAdded == 0 {
		t.Errorf("wanted snapshot data added to be non-zero, got: %v", snapshot.SnapshotSummary.DataAdded)
	}
	if snapshot.SnapshotSummary.TotalFilesProcessed == 0 {
		t.Errorf("wanted snapshot total files processed to be non-zero, got: %v", snapshot.SnapshotSummary.TotalFilesProcessed)
	}
	if snapshot.SnapshotSummary.TotalBytesProcessed == 0 {
		t.Errorf("wanted snapshot total bytes processed to be non-zero, got: %v", snapshot.SnapshotSummary.TotalBytesProcessed)
	}
	if snapshot.SnapshotSummary.DurationMs() == 0 {
		t.Errorf("wanted snapshot duration to be non-zero, got: %v", snapshot.SnapshotSummary.DurationMs())
	}
}

func TestLs(t *testing.T) {
	t.Parallel()

	repo := t.TempDir()
	r := NewRepo(helpers.ResticBinary(t), repo, WithFlags("--no-cache"), WithEnv("RESTIC_PASSWORD=test"))
	if err := r.Init(context.Background()); err != nil {
		t.Fatalf("failed to init repo: %v", err)
	}

	testData := helpers.CreateTestData(t)

	snapshot, err := r.Backup(context.Background(), []string{testData}, nil)
	if err != nil {
		t.Fatalf("failed to backup and create new snapshot: %v", err)
	}

	_, entries, err := r.ListDirectory(context.Background(), snapshot.SnapshotId, toRepoPath(testData))

	if err != nil {
		t.Fatalf("failed to list directory: %v", err)
	}

	if len(entries) != 101 {
		t.Errorf("wanted 101 entries, got: %d", len(entries))
	}
}

func TestResticForget(t *testing.T) {
	t.Parallel()

	repo := t.TempDir()
	r := NewRepo(helpers.ResticBinary(t), repo, WithFlags("--no-cache"), WithEnv("RESTIC_PASSWORD=test"))
	if err := r.Init(context.Background()); err != nil {
		t.Fatalf("failed to init repo: %v", err)
	}

	testData := helpers.CreateTestData(t)

	ids := make([]string, 0)
	for i := 0; i < 10; i++ {
		output, err := r.Backup(context.Background(), []string{testData}, nil)
		if err != nil {
			t.Fatalf("failed to backup and create new snapshot: %v", err)
		}
		ids = append(ids, output.SnapshotId)
	}

	// forget snapshots
	res, err := r.Forget(context.Background(), &RetentionPolicy{KeepLastN: 3})
	if err != nil {
		t.Fatalf("failed to forget snapshots: %v", err)
	}

	if len(res.Keep) != 3 {
		t.Errorf("wanted 3 snapshots to be kept, got: %d", len(res.Keep))
	}

	if len(res.Remove) != 7 {
		t.Errorf("wanted 7 snapshots to be removed, got: %d", len(res.Remove))
	}

	removedIds := make([]string, 0)
	for _, snapshot := range res.Remove {
		removedIds = append(removedIds, snapshot.Id)
	}
	slices.Reverse(removedIds)
	keptIds := make([]string, 0)
	for _, snapshot := range res.Keep {
		keptIds = append(keptIds, snapshot.Id)
	}
	slices.Reverse(keptIds)

	if !reflect.DeepEqual(removedIds, ids[:7]) {
		t.Errorf("wanted removed ids to be %v, got: %v", ids[:7], removedIds)
	}

	if !reflect.DeepEqual(keptIds, ids[7:]) {
		t.Errorf("wanted kept ids to be %v, got: %v", ids[7:], keptIds)
	}
}

func TestForgetSnapshotId(t *testing.T) {
	t.Parallel()

	repo := t.TempDir()
	r := NewRepo(helpers.ResticBinary(t), repo, WithFlags("--no-cache"), WithEnv("RESTIC_PASSWORD=test"))
	if err := r.Init(context.Background()); err != nil {
		t.Fatalf("failed to init repo: %v", err)
	}

	testData := helpers.CreateTestData(t)

	ids := make([]string, 0)
	for i := 0; i < 5; i++ {
		output, err := r.Backup(context.Background(), []string{testData}, nil)
		if err != nil {
			t.Fatalf("failed to backup and create new snapshot: %v", err)
		}
		ids = append(ids, output.SnapshotId)
	}

	// forget snapshot by ID
	err := r.ForgetSnapshot(context.Background(), ids[0])
	if err != nil {
		t.Fatalf("failed to forget snapshots: %v", err)
	}

	snapshots, err := r.Snapshots(context.Background())
	if err != nil {
		t.Fatalf("failed to list snapshots: %v", err)
	}
	if len(snapshots) != 4 {
		t.Errorf("wanted 4 snapshots, got: %d", len(snapshots))
	}
}

func TestResticPrune(t *testing.T) {
	t.Parallel()

	repo := t.TempDir()
	r := NewRepo(helpers.ResticBinary(t), repo, WithFlags("--no-cache"), WithEnv("RESTIC_PASSWORD=test"))
	if err := r.Init(context.Background()); err != nil {
		t.Fatalf("failed to init repo: %v", err)
	}

	testData := helpers.CreateTestData(t)

	for i := 0; i < 3; i++ {
		_, err := r.Backup(context.Background(), []string{testData}, nil)
		if err != nil {
			t.Fatalf("failed to backup: %v", err)
		}
	}

	// forget recent snapshots
	_, err := r.Forget(context.Background(), &RetentionPolicy{KeepLastN: 1})
	if err != nil {
		t.Fatalf("failed to forget snapshots: %v", err)
	}

	// prune all snapshots
	output := bytes.NewBuffer(nil)
	if err := r.Prune(context.Background(), output); err != nil {
		t.Fatalf("failed to prune snapshots: %v", err)
	}

	wantStr := "collecting packs for deletion and repacking"

	if !bytes.Contains(output.Bytes(), []byte(wantStr)) {
		t.Errorf("wanted output to contain 'keep 1 snapshots', got: %s", output.String())
	}
}

func TestResticRestore(t *testing.T) {
	t.Parallel()

	repo := t.TempDir()
	r := NewRepo(helpers.ResticBinary(t), repo, WithFlags("--no-cache"), WithEnv("RESTIC_PASSWORD=test"))
	if err := r.Init(context.Background()); err != nil {
		t.Fatalf("failed to init repo: %v", err)
	}

	restorePath := t.TempDir()

	testData := helpers.CreateTestData(t)
	dirCount := strings.Count(testData, string(filepath.Separator))
	if runtime.GOOS == "windows" {
		// On Windows, the volume name is also included as a dir in the path.
		dirCount += 1
	}

	snapshot, err := r.Backup(context.Background(), []string{testData}, nil)
	if err != nil {
		t.Fatalf("failed to backup and create new snapshot: %v", err)
	}

	// restore all files
	summary, err := r.Restore(context.Background(), snapshot.SnapshotId, func(event *RestoreProgressEntry) {
		t.Logf("restore event: %v", event)
	}, WithFlags("--target", restorePath))
	if err != nil {
		t.Fatalf("failed to restore snapshot: %v", err)
	}

	// should be 100 files + parent directories.
	fileCount := 100 + dirCount
	if summary.TotalFiles != int64(fileCount) {
		t.Errorf("wanted %d files to be restored, got: %d", fileCount, summary.TotalFiles)
	}
}

func TestResticStats(t *testing.T) {
	t.Parallel()

	repo := t.TempDir()
	r := NewRepo(helpers.ResticBinary(t), repo, WithFlags("--no-cache"), WithEnv("RESTIC_PASSWORD=test"))
	if err := r.Init(context.Background()); err != nil {
		t.Fatalf("failed to init repo: %v", err)
	}

	testData := helpers.CreateTestData(t)

	_, err := r.Backup(context.Background(), []string{testData}, nil)
	if err != nil {
		t.Fatalf("failed to backup and create new snapshot: %v", err)
	}

	// restore all files
	stats, err := r.Stats(context.Background())
	if err != nil {
		t.Fatalf("failed to get stats: %v", err)
	}
	if stats.SnapshotsCount != 1 {
		t.Errorf("wanted 1 snapshot, got: %d", stats.SnapshotsCount)
	}
	if stats.TotalSize == 0 {
		t.Errorf("wanted non-zero total size, got: %d", stats.TotalSize)
	}
	if stats.TotalUncompressedSize == 0 {
		t.Errorf("wanted non-zero total uncompressed size, got: %d", stats.TotalUncompressedSize)
	}
	if stats.TotalBlobCount == 0 {
		t.Errorf("wanted non-zero total blob count, got: %d", stats.TotalBlobCount)
	}
}

func TestResticCheck(t *testing.T) {
	t.Parallel()

	repo := t.TempDir()
	r := NewRepo(helpers.ResticBinary(t), repo, WithFlags("--no-cache"), WithEnv("RESTIC_PASSWORD=test"))
	if err := r.Init(context.Background()); err != nil {
		t.Fatalf("failed to init repo: %v", err)
	}

	testData := helpers.CreateTestData(t)

	_, err := r.Backup(context.Background(), []string{testData}, nil)
	if err != nil {
		t.Fatalf("failed to backup and create new snapshot: %v", err)
	}

	// check repo
	output := bytes.NewBuffer(nil)
	if err := r.Check(context.Background(), output, WithFlags("--read-data")); err != nil {
		t.Fatalf("failed to check repo: %v", err)
	}

	wantStr := "no errors were found"
	if !bytes.Contains(output.Bytes(), []byte(wantStr)) {
		t.Errorf("wanted output to contain 'no errors were found', got: %s", output.String())
	}
}

func toRepoPath(path string) string {
	if runtime.GOOS != "windows" {
		return path
	}

	// On Windows, the temp directory path needs to be converted to a repo path
	// for restic to interpret it correctly in restore/snapshot operations.
	sepIdx := strings.Index(path, string(filepath.Separator))
	if sepIdx != 2 || path[1] != ':' {
		return path
	}
	return filepath.ToSlash(filepath.Join(
		string(filepath.Separator), // leading slash
		string(path[0]),            // drive volume
		path[3:],                   // path
	))
}

func BenchmarkBackup(t *testing.B) {
	repo := t.TempDir()
	r := NewRepo(helpers.ResticBinary(t), repo, WithFlags("--no-cache"), WithEnv("RESTIC_PASSWORD=test"))
	if err := r.Init(context.Background()); err != nil {
		t.Fatalf("failed to init repo: %v", err)
	}

	workdir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	t.ResetTimer()

	for i := 0; i < t.N; i++ {
		_, err := r.Backup(context.Background(), []string{workdir}, func(e *BackupProgressEntry) {})
		if err != nil {
			t.Fatalf("failed to backup: %v", err)
		}
	}
}

func BenchmarkBackupWithSimulatedCallback(t *testing.B) {
	repo := t.TempDir()
	r := NewRepo(helpers.ResticBinary(t), repo, WithFlags("--no-cache"), WithEnv("RESTIC_PASSWORD=test"))
	if err := r.Init(context.Background()); err != nil {
		t.Fatalf("failed to init repo: %v", err)
	}

	workdir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	t.ResetTimer()

	for i := 0; i < t.N; i++ {
		_, err := r.Backup(context.Background(), []string{workdir}, func(e *BackupProgressEntry) {
			time.Sleep(50 * time.Millisecond) // simulate work being done in the callback
		})
		if err != nil {
			t.Fatalf("failed to backup: %v", err)
		}
	}
}
