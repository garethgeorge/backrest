package repo

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/test/helpers"
	test "github.com/garethgeorge/backrest/test/helpers"
	"golang.org/x/sync/errgroup"
)

var configForTest = &v1.Config{
	Instance: "test",
}

func TestBackup(t *testing.T) {
	t.Parallel()

	testData := test.CreateTestData(t)

	tcs := []struct {
		name        string
		repo        *v1.Repo
		plan        *v1.Plan
		excludeGoos []string
	}{
		{
			name: "backup",
			repo: &v1.Repo{
				Id:       "test",
				Uri:      t.TempDir(),
				Password: "test",
			},
			plan: &v1.Plan{
				Id:    "test",
				Repo:  "test",
				Paths: []string{testData},
			},
		},
		{
			name: "backup with ionice",
			repo: &v1.Repo{
				Id:       "test",
				Uri:      t.TempDir(),
				Password: "test",
				CommandPrefix: &v1.CommandPrefix{
					IoNice:  v1.CommandPrefix_IO_BEST_EFFORT_LOW,
					CpuNice: v1.CommandPrefix_CPU_LOW,
				},
			},
			plan: &v1.Plan{
				Id:    "test",
				Repo:  "test",
				Paths: []string{testData},
			},
			excludeGoos: []string{"windows", "darwin"},
		},
	}

	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if slices.Contains(tc.excludeGoos, runtime.GOOS) {
				t.Skipf("skipping test on %s", runtime.GOOS)
			}

			orchestrator := initRepoHelper(t, configForTest, tc.repo)

			summary, err := orchestrator.Backup(context.Background(), tc.plan, false, nil)
			if err != nil {
				t.Fatalf("backup error: %v", err)
			}

			if summary.SnapshotId == "" {
				t.Fatal("expected snapshot id")
			}

			if summary.FilesNew != 100 {
				t.Fatalf("expected 100 new files, got %d", summary.FilesNew)
			}
		})
	}
}

func TestRestore(t *testing.T) {
	t.Parallel()

	// Use a filepath that exercises a few of the glob characters to test escaping
	messyFilePathToTestGlobs := "test.txt"
	if runtime.GOOS != "windows" {
		messyFilePathToTestGlobs = "test*?[].txt"
	}

	testFile := path.Join(t.TempDir(), messyFilePathToTestGlobs)
	if err := ioutil.WriteFile(testFile, []byte("lorum ipsum"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	r := &v1.Repo{
		Id:       "test",
		Uri:      t.TempDir(),
		Password: "test",
		Flags:    []string{"--no-cache"},
	}

	plan := &v1.Plan{
		Id:    "test",
		Repo:  "test",
		Paths: []string{testFile},
	}

	orchestrator := initRepoHelper(t, configForTest, r)

	// Create a backup of the single file
	summary, err := orchestrator.Backup(context.Background(), plan, false, nil)
	if err != nil {
		t.Fatalf("backup error: %v", err)
	}
	if summary.SnapshotId == "" {
		t.Fatal("expected snapshot id")
	}
	if summary.FilesNew != 1 {
		t.Fatalf("expected 1 new file, got %d", summary.FilesNew)
	}

	// Restore the file
	restoreDir := t.TempDir()
	snapshotPath := strings.ReplaceAll(testFile, ":", "") // remove the colon from the windows path e.g. C:\test.txt -> C\test.txt
	restoreSummary, err := orchestrator.Restore(context.Background(), summary.SnapshotId, snapshotPath, restoreDir, nil)
	if err != nil {
		t.Fatalf("restore error: %v", err)
	}
	t.Logf("restore summary: %+v", restoreSummary)

	if runtime.GOOS == "windows" {
		return
	}

	if restoreSummary.FilesRestored != 1 {
		t.Errorf("expected 1 new file, got %d", restoreSummary.FilesRestored)
	}
	if restoreSummary.TotalFiles != 1 {
		t.Errorf("expected 1 total file, got %d", restoreSummary.TotalFiles)
	}

	// Check the restored file
	restoredFile := path.Join(restoreDir, messyFilePathToTestGlobs)
	if _, err := os.Stat(restoredFile); err != nil {
		t.Fatalf("failed to stat restored file: %v", err)
	}
	restoredData, err := os.ReadFile(restoredFile)
	if err != nil {
		t.Fatalf("failed to read restored file: %v", err)
	}
	if string(restoredData) != "lorum ipsum" {
		t.Fatalf("expected 'test', got '%s'", restoredData)
	}
}

func TestSnapshotParenting(t *testing.T) {
	t.Parallel()

	repo := t.TempDir()
	testData := test.CreateTestData(t)

	// create a new repo with cache disabled for testing
	r := &v1.Repo{
		Id:       "test",
		Uri:      repo,
		Password: "test",
		Flags:    []string{"--no-cache"},
	}

	plans := []*v1.Plan{
		{
			Id:    "test",
			Repo:  "test",
			Paths: []string{testData},
		},
		{
			Id:    "test2",
			Repo:  "test",
			Paths: []string{testData},
		},
	}

	orchestrator := initRepoHelper(t, configForTest, r)

	var eg errgroup.Group
	for _, plan := range plans {
		eg.Go(func() error {
			for i := 0; i < 2; i++ {
				summary, err := orchestrator.Backup(context.Background(), plan, false, nil)
				if err != nil {
					return fmt.Errorf("failed to backup plan %s: %v", plan.Id, err)
				}

				if summary.SnapshotId == "" {
					return errors.New("expected snapshot id")
				}

				if summary.TotalFilesProcessed != 100 {
					t.Logf("summary is: %+v", summary)
					return fmt.Errorf("expected 100 done files, got %d", summary.TotalFilesProcessed)
				}
			}
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		t.Fatal(err)
	}

	for _, plan := range plans {
		t.Run("verify_"+plan.Id, func(t *testing.T) {
			t.Parallel()
			snapshots, err := orchestrator.SnapshotsForPlan(context.Background(), plan)
			if err != nil {
				t.Errorf("failed to get snapshots for plan %s: %v", plan.Id, err)
				return
			}

			if len(snapshots) != 2 {
				t.Errorf("expected 4 snapshots, got %d", len(snapshots))
			}

			for i := 1; i < len(snapshots); i++ {
				prev := snapshots[i-1]
				curr := snapshots[i]

				if prev.UnixTimeMs() >= curr.UnixTimeMs() {
					t.Errorf("snapshots are out of order")
				}

				if prev.Id != curr.Parent {
					t.Errorf("expected snapshot %s to have parent %s, got %s", curr.Id, prev.Id, curr.Parent)
				}

				if !slices.Contains(curr.Tags, TagForPlan(plan.Id)) {
					t.Errorf("expected snapshot %s to have tag %s", curr.Id, TagForPlan(plan.Id))
				}
			}
		})
	}

	t.Run("verify_snapshot_count", func(t *testing.T) {
		t.Parallel()
		snapshots, err := orchestrator.Snapshots(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if len(snapshots) != 4 {
			t.Errorf("expected 4 snapshots, got %d", len(snapshots))
		}
	})
}

func TestEnvVarPropagation(t *testing.T) {
	repo := t.TempDir()

	// create a new repo with cache disabled for testing
	r := &v1.Repo{
		Id:    "test",
		Uri:   repo,
		Flags: []string{"--no-cache"},
		Env:   []string{"RESTIC_PASSWORD=${MY_FOO}"},
	}

	orchestrator, err := NewRepoOrchestrator(configForTest, r, helpers.ResticBinary(t))
	if err != nil {
		t.Fatalf("failed to create repo orchestrator: %v", err)
	}

	err = orchestrator.Init(context.Background())
	if err == nil || !strings.Contains(err.Error(), "password") {
		t.Fatalf("expected error about RESTIC_PASSWORD, got: %v", err)
	}

	// set the env var
	os.Setenv("MY_FOO", "bar")
	defer os.Unsetenv("MY_FOO")
	orchestrator, err = NewRepoOrchestrator(configForTest, r, helpers.ResticBinary(t))
	if err != nil {
		t.Fatalf("failed to create repo orchestrator: %v", err)
	}

	err = orchestrator.Init(context.Background())
	if err != nil {
		t.Fatalf("backup error: %v", err)
	}
}

func TestCheck(t *testing.T) {
	t.Parallel()

	tcs := []struct {
		name string
		repo *v1.Repo
	}{
		{
			name: "check structure",
			repo: &v1.Repo{
				Id:       "test",
				Uri:      t.TempDir(),
				Password: "test",
				CheckPolicy: &v1.CheckPolicy{
					Mode: nil,
				},
			},
		},
		{
			name: "read data percent",
			repo: &v1.Repo{
				Id:       "test",
				Uri:      t.TempDir(),
				Password: "test",
				CheckPolicy: &v1.CheckPolicy{
					Mode: &v1.CheckPolicy_ReadDataSubsetPercent{
						ReadDataSubsetPercent: 50,
					},
				},
			},
		},
	}

	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			orchestrator := initRepoHelper(t, configForTest, tc.repo)
			buf := bytes.NewBuffer(nil)

			err := orchestrator.Init(context.Background())
			if err != nil {
				t.Fatalf("init error: %v", err)
			}

			err = orchestrator.Check(context.Background(), buf)
			if err != nil {
				t.Errorf("check error: %v", err)
			}
			t.Logf("check output: %s", buf.String())
		})
	}
}

func initRepoHelper(t *testing.T, config *v1.Config, repo *v1.Repo) *RepoOrchestrator {
	orchestrator, err := NewRepoOrchestrator(config, repo, helpers.ResticBinary(t))
	if err != nil {
		t.Fatalf("failed to create repo orchestrator: %v", err)
	}

	err = orchestrator.Init(context.Background())
	if err != nil {
		t.Fatalf("init error: %v", err)
	}

	return orchestrator
}

func TestRestoreAmbiguity(t *testing.T) {
	t.Parallel()
	repoDir := t.TempDir()
	sourceDir := t.TempDir()

	targetDir := filepath.Join(sourceDir, "target")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := ioutil.WriteFile(filepath.Join(targetDir, "target.txt"), []byte("content1"), 0644); err != nil {
		t.Fatal(err)
	}

	nestedTargetDir := filepath.Join(sourceDir, "nested", "target")
	if err := os.MkdirAll(nestedTargetDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := ioutil.WriteFile(filepath.Join(nestedTargetDir, "target.txt"), []byte("content2"), 0644); err != nil {
		t.Fatal(err)
	}

	r := &v1.Repo{
		Id:       "test-restore-ambiguity",
		Uri:      repoDir,
		Password: "test",
		Flags:    []string{"--no-cache"},
	}

	plan := &v1.Plan{
		Id:    "test-plan",
		Repo:  "test-restore-ambiguity",
		Paths: []string{sourceDir},
	}

	orchestrator := initRepoHelper(t, configForTest, r)

	summary, err := orchestrator.Backup(context.Background(), plan, false, nil)
	if err != nil {
		t.Fatalf("backup error: %v", err)
	}

	// Restore "target"
	restoreDir := t.TempDir()

	// We pass the full path as it appears in the snapshot.
	// On Windows, absolute paths like C:\Users are stored as /C/Users.
	restorePath := sourceDir
	if runtime.GOOS == "windows" {
		vol := filepath.VolumeName(restorePath)
		if vol != "" {
			restorePath = "/" + strings.TrimSuffix(vol, ":") + filepath.ToSlash(restorePath[len(vol):])
		} else {
			restorePath = filepath.ToSlash(restorePath)
		}
	}
	restorePath = path.Join(restorePath, "target")

	restoreSummary, err := orchestrator.Restore(context.Background(), summary.SnapshotId, restorePath, restoreDir, nil)
	if err != nil {
		t.Fatalf("restore error: %v", err)
	}

	t.Logf("Restore summary: %+v", restoreSummary)

	// Since we restored with a subtree root, the files are relative to that root.
	nestedFile := filepath.Join(restoreDir, "nested", "target", "target.txt")
	if _, err := os.Stat(nestedFile); err == nil {
		t.Errorf("FAIL: Found unexpected file from nested duplicate folder: %s", nestedFile)
	} else if !os.IsNotExist(err) {
		t.Errorf("Error checking for nested file: %v", err)
	}

	// Verify the correct one exists
	expectedFile := filepath.Join(restoreDir, "target", "target.txt")
	if _, err := os.Stat(expectedFile); err != nil {
		t.Errorf("FAIL: Expected main file missing: %s", expectedFile)
	}
}
