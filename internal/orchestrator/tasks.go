package orchestrator

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/garethgeorge/resticui/gen/go/v1"
	"github.com/garethgeorge/resticui/internal/database/oplog"
	"github.com/garethgeorge/resticui/pkg/restic"
	"github.com/gitploy-io/cronexpr"
	"github.com/hashicorp/go-multierror"
	"go.uber.org/zap"
)


type Task interface {
	Name() string // huamn readable name for this task.
	Next(now time.Time) *time.Time // when this task would like to be run.
	Run(ctx context.Context) error // run the task.
}

// BackupTask is a scheduled backup operation.
type ScheduledBackupTask struct {
	orchestrator *Orchestrator // owning orchestrator
	plan *v1.Plan
	schedule *cronexpr.Schedule 
}

var _ Task = &ScheduledBackupTask{}

func NewScheduledBackupTask(orchestrator *Orchestrator, plan *v1.Plan) (*ScheduledBackupTask, error) {
	sched, err := cronexpr.Parse(plan.Cron)
	if err != nil {
		return nil, fmt.Errorf("failed to parse schedule %q: %w", plan.Cron, err)
	}

	return &ScheduledBackupTask{
		orchestrator: orchestrator,
		plan: plan,
		schedule: sched,
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
	plan *v1.Plan
	time *time.Time
}

func NewOneofBackupTask(orchestrator *Orchestrator, plan *v1.Plan, at time.Time) *OnetimeBackupTask {
	return &OnetimeBackupTask{
		orchestrator: orchestrator,
		plan: plan,
		time: &at,
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
		PlanId: plan.Id,
		RepoId: plan.Repo,
		UnixTimeStartMs: curTimeMillis(),
		Status: v1.OperationStatus_STATUS_INPROGRESS,
		Op: backupOp,
	}

	return WithOperation(orchestrator.oplog, op, func() error {
		zap.L().Info("Starting backup", zap.String("plan", plan.Id))
		repo, err := orchestrator.GetRepo(plan.Repo)
		if err != nil {
			return fmt.Errorf("failed to get repo %q: %w", plan.Repo, err)
		}

		lastSent := time.Now() // debounce progress updates, these can endup being very frequent.
		summary, err := repo.Backup(ctx, plan, func(entry *restic.BackupProgressEntry) {
			if time.Since(lastSent) < 200 * time.Millisecond {
				return
			}
			lastSent = time.Now()

			backupOp.OperationBackup.LastStatus = entry.ToProto()
			if err := orchestrator.oplog.Update(op); err != nil {
				zap.S().Errorf("failed to update oplog with progress for backup: %v", err)
			}
			zap.L().Debug("Backup progress", zap.Float64("progress", entry.PercentDone))
		})
		if err != nil {
			return fmt.Errorf("failed to backup repo %q: %w", plan.Repo, err)
		}

		backupOp.OperationBackup.LastStatus = summary.ToProto()
		if err := orchestrator.oplog.Update(op); err != nil {
			return fmt.Errorf("update oplog with summary for backup: %v", err)
		}

		zap.L().Info("Backup complete", zap.String("plan", plan.Id))
		return nil
	})
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
	if e := oplog.Update(op); err != nil {
		return multierror.Append(err, fmt.Errorf("failed to update operation in oplog: %w", e))
	}
	return err
}

func curTimeMillis() int64 {
	t := time.Now()
	return t.Unix() * 1000 + int64(t.Nanosecond() / 1000000)
}