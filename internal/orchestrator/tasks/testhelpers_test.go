package tasks

import (
	"context"
	"io"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/pkg/restic"
)

// fakeRepoOrchestrator is a test double for the RepoOrchestrator interface.
// Each method returns the corresponding configured result/error fields.
type fakeRepoOrchestrator struct {
	unlockErr error

	backupResult *restic.BackupProgressEntry
	backupErr    error

	forgetResult []*v1.ResticSnapshot
	forgetErr    error

	forgetSnapshotErr error

	pruneErr error
	checkErr error

	statsResult *v1.RepoStats
	statsErr    error

	restoreResult *v1.RestoreProgressEntry
	restoreErr    error

	snapshots    []*restic.Snapshot
	snapshotsErr error

	addTagsErr error

	runCommandErr error
}

var _ RepoOrchestrator = &fakeRepoOrchestrator{}

func (f *fakeRepoOrchestrator) UnlockIfAutoEnabled(ctx context.Context) error {
	return f.unlockErr
}

func (f *fakeRepoOrchestrator) Backup(ctx context.Context, plan *v1.Plan, dryRun bool, cb func(event *restic.BackupProgressEntry)) (*restic.BackupProgressEntry, error) {
	if cb != nil && f.backupResult != nil {
		cb(f.backupResult)
	}
	return f.backupResult, f.backupErr
}

func (f *fakeRepoOrchestrator) Forget(ctx context.Context, policy *v1.RetentionPolicy, opts ...restic.GenericOption) ([]*v1.ResticSnapshot, error) {
	return f.forgetResult, f.forgetErr
}

func (f *fakeRepoOrchestrator) ForgetSnapshot(ctx context.Context, snapshotId string) error {
	return f.forgetSnapshotErr
}

func (f *fakeRepoOrchestrator) Prune(ctx context.Context, output io.Writer) error {
	return f.pruneErr
}

func (f *fakeRepoOrchestrator) Check(ctx context.Context, output io.Writer) error {
	return f.checkErr
}

func (f *fakeRepoOrchestrator) Stats(ctx context.Context) (*v1.RepoStats, error) {
	return f.statsResult, f.statsErr
}

func (f *fakeRepoOrchestrator) Restore(ctx context.Context, snapshotId string, snapshotPath string, target string, cb func(event *v1.RestoreProgressEntry)) (*v1.RestoreProgressEntry, error) {
	if cb != nil && f.restoreResult != nil {
		cb(f.restoreResult)
	}
	return f.restoreResult, f.restoreErr
}

func (f *fakeRepoOrchestrator) Snapshots(ctx context.Context) ([]*restic.Snapshot, error) {
	return f.snapshots, f.snapshotsErr
}

func (f *fakeRepoOrchestrator) AddTags(ctx context.Context, snapshotIDs []string, tags []string) error {
	return f.addTagsErr
}

func (f *fakeRepoOrchestrator) RunCommand(ctx context.Context, command string, writer io.Writer) error {
	return f.runCommandErr
}
