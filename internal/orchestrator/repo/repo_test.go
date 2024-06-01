package repo

import (
	"bytes"
	"context"
	"os"
	"runtime"
	"slices"
	"strings"
	"testing"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/test/helpers"
	test "github.com/garethgeorge/backrest/test/helpers"
)

var configForTest = &v1.Config{
	Instance: "test",
}

func TestBackup(t *testing.T) {
	t.Parallel()

	testData := test.CreateTestData(t)

	tcs := []struct {
		name     string
		repo     *v1.Repo
		plan     *v1.Plan
		unixOnly bool
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
			unixOnly: true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			if tc.unixOnly && runtime.GOOS == "windows" {
				t.Skip("skipping on windows")
			}

			orchestrator, err := NewRepoOrchestrator(configForTest, tc.repo, helpers.ResticBinary(t))
			if err != nil {
				t.Fatalf("failed to create repo orchestrator: %v", err)
			}

			summary, err := orchestrator.Backup(context.Background(), tc.plan, nil)
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

	orchestrator, err := NewRepoOrchestrator(configForTest, r, helpers.ResticBinary(t))
	if err != nil {
		t.Fatalf("failed to create repo orchestrator: %v", err)
	}

	for i := 0; i < 4; i++ {
		for _, plan := range plans {
			summary, err := orchestrator.Backup(context.Background(), plan, nil)
			if err != nil {
				t.Fatalf("failed to backup plan %s: %v", plan.Id, err)
			}

			if summary.SnapshotId == "" {
				t.Errorf("expected snapshot id")
			}

			if summary.TotalFilesProcessed != 100 {
				t.Logf("summary is: %+v", summary)
				t.Errorf("expected 100 done files, got %d", summary.TotalFilesProcessed)
			}
		}
	}

	for _, plan := range plans {
		snapshots, err := orchestrator.SnapshotsForPlan(context.Background(), plan)
		if err != nil {
			t.Errorf("failed to get snapshots for plan %s: %v", plan.Id, err)
			continue
		}

		if len(snapshots) != 4 {
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
	}

	snapshots, err := orchestrator.Snapshots(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshots) != 8 {
		t.Errorf("expected 8 snapshots, got %d", len(snapshots))
	}
}

func TestEnvVarPropagation(t *testing.T) {
	t.Parallel()

	repo := t.TempDir()
	testData := test.CreateTestData(t)

	// create a new repo with cache disabled for testing
	r := &v1.Repo{
		Id:    "test",
		Uri:   repo,
		Flags: []string{"--no-cache"},
		Env:   []string{"RESTIC_PASSWORD=${MY_FOO}"},
	}

	plan := &v1.Plan{
		Id:    "test",
		Repo:  "test",
		Paths: []string{testData},
	}

	orchestrator, err := NewRepoOrchestrator(configForTest, r, helpers.ResticBinary(t))
	if err != nil {
		t.Fatalf("failed to create repo orchestrator: %v", err)
	}

	_, err = orchestrator.Backup(context.Background(), plan, nil)
	if err == nil || !strings.Contains(err.Error(), "password") {
		t.Fatalf("expected error about RESTIC_PASSWORD, got: %v", err)
	}

	// set the env var
	os.Setenv("MY_FOO", "bar")
	orchestrator, err = NewRepoOrchestrator(configForTest, r, helpers.ResticBinary(t))
	if err != nil {
		t.Fatalf("failed to create repo orchestrator: %v", err)
	}

	summary, err := orchestrator.Backup(context.Background(), plan, nil)
	if err != nil {
		t.Fatalf("backup error: %v", err)
	}

	if summary.SnapshotId == "" {
		t.Fatal("expected snapshot id")
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
		t.Run(tc.name, func(t *testing.T) {
			orchestrator, err := NewRepoOrchestrator(configForTest, tc.repo, helpers.ResticBinary(t))
			if err != nil {
				t.Fatalf("failed to create repo orchestrator: %v", err)
			}

			buf := bytes.NewBuffer(nil)

			err = orchestrator.Init(context.Background())
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
