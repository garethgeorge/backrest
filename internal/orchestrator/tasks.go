package orchestrator

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/garethgeorge/resticui/gen/go/v1"
	"github.com/garethgeorge/resticui/internal/oplog"
	"github.com/garethgeorge/resticui/pkg/restic"
	"github.com/gitploy-io/cronexpr"
	"github.com/hashicorp/go-multierror"
	"go.uber.org/zap"
)


type task interface {
	Name() string // huamn readable name for this task.
	Next(now time.Time) *time.Time // when this task would like to be run.
	Run(ctx context.Context) error // run the task.
}

type backupTask struct {
	orchestrator *Orchestrator // owning orchestrator
	plan *v1.Plan
	schedule *cronexpr.Schedule 
}

var _ task = &backupTask{}

func newBackupTask(orchestrator *Orchestrator, plan *v1.Plan) (*backupTask, error) {
	sched, err := cronexpr.Parse(plan.Cron)
	if err != nil {
		return nil, fmt.Errorf("failed to parse schedule %q: %w", plan.Cron, err)
	}

	return &backupTask{
		orchestrator: orchestrator,
		plan: plan,
		schedule: sched,
	}, nil
}

func (t *backupTask) Name() string {
	return fmt.Sprintf("execute backup plan %v", t.plan.Id)
}

func (t *backupTask) Next(now time.Time) *time.Time {
	next := t.schedule.Next(now)
	return &next
}

func (t *backupTask) Run(ctx context.Context) error {
	backupOp := &v1.Operation_OperationBackup{
		OperationBackup: &v1.OperationBackup{},
	}

	op := &v1.Operation{
		PlanId: t.plan.Id,
		RepoId: t.plan.Repo,
		UnixTimeStartMs: time.Now().Unix(),
		Status: v1.OperationStatus_STATUS_INPROGRESS,
		Op: backupOp,
	}

	return WithOperation(t.orchestrator.oplog, op, func() error {
		zap.L().Info("Starting backup", zap.String("plan", t.plan.Id))
		repo, err := t.orchestrator.GetRepo(t.plan.Repo)
		if err != nil {
			return fmt.Errorf("failed to get repo %q: %w", t.plan.Repo, err)
		}

		if _, err := repo.Backup(ctx, t.plan, func(entry *restic.BackupProgressEntry) {
			backupOp.OperationBackup.LastStatus = entry.ToProto()
			if err := t.orchestrator.oplog.Update(op); err != nil {
				zap.S().Errorf("failed to update oplog with progress for backup: %v", err)
			}
			zap.L().Info("Backup progress", zap.Float64("progress", entry.PercentDone))
		}); err != nil {
			return fmt.Errorf("failed to backup repo %q: %w", t.plan.Repo, err)
		}

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
	op.UnixTimeEndMs = time.Now().Unix()
	if op.Status == v1.OperationStatus_STATUS_INPROGRESS {
		op.Status = v1.OperationStatus_STATUS_SUCCESS
	}
	if e := oplog.Update(op); err != nil {
		return multierror.Append(err, fmt.Errorf("failed to update operation in oplog: %w", e))
	}
	return err
}