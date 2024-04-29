package repo

import (
	"context"
	"slices"
	"testing"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/test/helpers"
	test "github.com/garethgeorge/backrest/test/helpers"
)

func TestBackup(t *testing.T) {
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

	plan := &v1.Plan{
		Id:    "test",
		Repo:  "test",
		Paths: []string{testData},
	}

	orchestrator, err := NewRepoOrchestrator(r, helpers.ResticBinary(t))
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

	if summary.FilesNew != 100 {
		t.Fatalf("expected 100 new files, got %d", summary.FilesNew)
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

	orchestrator, err := NewRepoOrchestrator(r, helpers.ResticBinary(t))
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

			if !slices.Contains(curr.Tags, tagForPlan(plan)) {
				t.Errorf("expected snapshot %s to have tag %s", curr.Id, tagForPlan(plan))
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
