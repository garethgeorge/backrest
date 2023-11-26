package orchestrator

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/garethgeorge/resticui/gen/go/v1"
	"github.com/garethgeorge/resticui/internal/oplog"
	"github.com/garethgeorge/resticui/internal/oplog/indexutil"
	"github.com/garethgeorge/resticui/internal/protoutil"
	"github.com/garethgeorge/resticui/pkg/restic"
	"github.com/gitploy-io/cronexpr"
	"github.com/hashicorp/go-multierror"
	"go.uber.org/zap"
)

type Task interface {
	Name() string                  // huamn readable name for this task.
	Next(now time.Time) *time.Time // when this task would like to be run.
	Run(ctx context.Context) error // run the task.
}

// BackupTask is a scheduled backup operation.
type ScheduledBackupTask struct {
	orchestrator *Orchestrator // owning orchestrator
	plan         *v1.Plan
	schedule     *cronexpr.Schedule
}

var _ Task = &ScheduledBackupTask{}

func NewScheduledBackupTask(orchestrator *Orchestrator, plan *v1.Plan) (*ScheduledBackupTask, error) {
	sched, err := cronexpr.ParseInLocation(plan.Cron, time.Now().Location().String())
	if err != nil {
		return nil, fmt.Errorf("failed to parse schedule %q: %w", plan.Cron, err)
	}

	return &ScheduledBackupTask{
		orchestrator: orchestrator,
		plan:         plan,
		schedule:     sched,
	}, nil
}

func (t *ScheduledBackupTask) Name() string {
	return fmt.Sprintf("backup for plan %q", t.plan.Id)
}

func (t *ScheduledBackupTask) Next(now time.Time) *time.Time {
	next := t.schedule.Next(now)
	return &next
}

func (t *ScheduledBackupTask) Run(ctx context.Context) error {
	return backupHelper(ctx, t.orchestrator, t.plan)
}

// OnetimeBackupTask is a single backup operation.
type OnetimeBackupTask struct {
	orchestrator *Orchestrator
	plan         *v1.Plan
	time         *time.Time
}

func NewOneofBackupTask(orchestrator *Orchestrator, plan *v1.Plan, at time.Time) *OnetimeBackupTask {
	return &OnetimeBackupTask{
		orchestrator: orchestrator,
		plan:         plan,
		time:         &at,
	}
}

func (t *OnetimeBackupTask) Name() string {
	return fmt.Sprintf("onetime backup for plan %q", t.plan.Id)
}

func (t *OnetimeBackupTask) Next(now time.Time) *time.Time {
	ret := t.time
	t.time = nil
	return ret
}

func (t *OnetimeBackupTask) Run(ctx context.Context) error {
	return backupHelper(ctx, t.orchestrator, t.plan)
}

// backupHelper does a backup.
func backupHelper(ctx context.Context, orchestrator *Orchestrator, plan *v1.Plan) error {
	backupOp := &v1.Operation_OperationBackup{
		OperationBackup: &v1.OperationBackup{},
	}

	op := &v1.Operation{
		PlanId:          plan.Id,
		RepoId:          plan.Repo,
		UnixTimeStartMs: curTimeMillis(),
		Status:          v1.OperationStatus_STATUS_INPROGRESS,
		Op:              backupOp,
	}

	startTime := time.Now()

	err := WithOperation(orchestrator.OpLog, op, func() error {
		zap.L().Info("Starting backup", zap.String("plan", plan.Id), zap.Int64("opId", op.Id))
		repo, err := orchestrator.GetRepo(plan.Repo)
		if err != nil {
			return fmt.Errorf("couldn't get repo %q: %w", plan.Repo, err)
		}

		lastSent := time.Now() // debounce progress updates, these can endup being very frequent.
		summary, err := repo.Backup(ctx, plan, func(entry *restic.BackupProgressEntry) {
			if time.Since(lastSent) < 250*time.Millisecond {
				return
			}
			lastSent = time.Now()

			backupOp.OperationBackup.LastStatus = protoutil.BackupProgressEntryToProto(entry)
			if err := orchestrator.OpLog.Update(op); err != nil {
				zap.S().Errorf("failed to update oplog with progress for backup: %v", err)
			}
			zap.L().Debug("backup progress", zap.Float64("progress", entry.PercentDone))
		})
		if err != nil {
			return fmt.Errorf("repo.Backup for repo %q: %w", plan.Repo, err)
		}

		op.SnapshotId = summary.SnapshotId
		backupOp.OperationBackup.LastStatus = protoutil.BackupProgressEntryToProto(summary)
		if backupOp.OperationBackup.LastStatus == nil {
			return fmt.Errorf("expected a final backup progress entry, got nil")
		}

		zap.L().Info("backup complete", zap.String("plan", plan.Id), zap.Duration("duration", time.Since(startTime)))
		return nil
	})
	if err != nil {
		return fmt.Errorf("backup operation: %w", err)
	}

	// this could alternatively be scheduled as a separate task, but it probably makes sense to index snapshots immediately after a backup.
	if err := indexSnapshotsHelper(ctx, orchestrator, plan); err != nil {
		return fmt.Errorf("reindexing snapshots after backup operation: %w", err)
	}

	return nil
}

func indexSnapshotsHelper(ctx context.Context, orchestrator *Orchestrator, plan *v1.Plan) error {
	repo, err := orchestrator.GetRepo(plan.Repo)
	if err != nil {
		return fmt.Errorf("couldn't get repo %q: %w", plan.Repo, err)
	}

	snapshots, err := repo.SnapshotsForPlan(ctx, plan)
	if err != nil {
		return fmt.Errorf("get snapshots for plan %q: %w", plan.Id, err)
	}

	startTime := time.Now()
	alreadyIndexed := 0
	var indexOps []*v1.Operation
	for _, snapshot := range snapshots {
		ops, err := orchestrator.OpLog.GetBySnapshotId(snapshot.Id, indexutil.CollectAll())
		if err != nil {
			return fmt.Errorf("HasIndexSnapshot for snapshot %q: %w", snapshot.Id, err)
		}

		if containsSnapshotOperation(ops) {
			alreadyIndexed += 1
			continue
		}

		snapshotProto := protoutil.SnapshotToProto(snapshot)
		indexOps = append(indexOps, &v1.Operation{
			RepoId:          plan.Repo,
			PlanId:          plan.Id,
			UnixTimeStartMs: snapshotProto.UnixTimeMs,
			UnixTimeEndMs:   snapshotProto.UnixTimeMs,
			Status:          v1.OperationStatus_STATUS_SUCCESS,
			SnapshotId:      snapshotProto.Id,
			Op: &v1.Operation_OperationIndexSnapshot{
				OperationIndexSnapshot: &v1.OperationIndexSnapshot{
					Snapshot: snapshotProto,
				},
			},
		})
	}

	if err := orchestrator.OpLog.BulkAdd(indexOps); err != nil {
		return fmt.Errorf("BulkAdd snapshot operations: %w", err)
	}

	zap.L().Debug("Indexed snapshots",
		zap.String("plan", plan.Id),
		zap.Duration("duration", time.Since(startTime)),
		zap.Int("alreadyIndexed", alreadyIndexed),
		zap.Int("newlyAdded", len(snapshots)-alreadyIndexed),
	)

	return err
}

// WithOperation is a utility that creates an operation to track the function's execution.
// timestamps are automatically added and the status is automatically updated if an error occurs.
func WithOperation(oplog *oplog.OpLog, op *v1.Operation, do func() error) error {
	if err := oplog.Add(op); err != nil {
		return fmt.Errorf("failed to add operation to oplog: %w", err)
	}
	if op.Status == v1.OperationStatus_STATUS_UNKNOWN {
		op.Status = v1.OperationStatus_STATUS_INPROGRESS
	}
	err := do()
	if err != nil {
		op.Status = v1.OperationStatus_STATUS_ERROR
		op.DisplayMessage = err.Error()
	}
	op.UnixTimeEndMs = curTimeMillis()
	if op.Status == v1.OperationStatus_STATUS_INPROGRESS {
		op.Status = v1.OperationStatus_STATUS_SUCCESS
	}
	if e := oplog.Update(op); e != nil {
		return multierror.Append(err, fmt.Errorf("failed to update operation in oplog: %w", e))
	}
	return err
}

func curTimeMillis() int64 {
	t := time.Now()
	return t.Unix()*1000 + int64(t.Nanosecond()/1000000)
}

func containsSnapshotOperation(ops []*v1.Operation) bool {
	for _, op := range ops {
		if _, ok := op.Op.(*v1.Operation_OperationIndexSnapshot); ok {
			return true
		}
	}
	return false
}
