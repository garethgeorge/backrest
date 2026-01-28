package tasks

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
)

// NewOneoffDryRunBackupTask creates a task that runs restic backup --dry-run -vv
// to validate backup configuration and estimate data volume without transferring data.
func NewOneoffDryRunBackupTask(repo *v1.Repo, planID string, flowID int64, at time.Time) Task {
	return &GenericOneoffTask{
		OneoffTask: OneoffTask{
			BaseTask: BaseTask{
				TaskType:   "dry_run_backup",
				TaskName:   fmt.Sprintf("dry run backup for plan %q", planID),
				TaskRepo:   repo,
				TaskPlanID: planID,
			},
			FlowID: flowID,
			RunAt:  at,
			ProtoOp: &v1.Operation{
				Op: &v1.Operation_OperationDryRunBackup{
					OperationDryRunBackup: &v1.OperationDryRunBackup{},
				},
			},
		},
		Do: func(ctx context.Context, st ScheduledTask, runner TaskRunner) error {
			op := st.Op
			dryRunOp, ok := op.Op.(*v1.Operation_OperationDryRunBackup)
			if !ok {
				return fmt.Errorf("unexpected operation type: %T", op.Op)
			}
			return dryRunBackupHelper(ctx, st, runner, dryRunOp.OperationDryRunBackup)
		},
	}
}

func dryRunBackupHelper(ctx context.Context, st ScheduledTask, runner TaskRunner, op *v1.OperationDryRunBackup) error {
	t := st.Task
	log := runner.Logger(ctx)

	// Get plan configuration
	plan, err := runner.GetPlan(t.PlanID())
	if err != nil {
		return fmt.Errorf("get plan %q: %w", t.PlanID(), err)
	}

	// Get repo orchestrator
	repo, err := runner.GetRepoOrchestrator(t.RepoID())
	if err != nil {
		return fmt.Errorf("get repo %q: %w", t.RepoID(), err)
	}

	// Create logref writer for capturing output
	logref, writer, err := runner.LogrefWriter()
	if err != nil {
		return fmt.Errorf("create logref writer: %w", err)
	}
	defer writer.Close()
	op.OutputLogref = logref

	log.Info("starting dry run backup")

	// Run dry run backup with verbose output
	result, err := repo.DryRunBackup(ctx, plan, writer)
	if err != nil {
		return fmt.Errorf("dry run backup: %w", err)
	}

	// Populate structured stats from parsed result
	if result != nil {
		op.FilesNew = result.FilesNew
		op.FilesChanged = result.FilesChanged
		op.FilesUnmodified = result.FilesUnmodified
		op.DirsNew = result.DirsNew
		op.DirsChanged = result.DirsChanged
		op.DirsUnmodified = result.DirsUnmodified
		op.DataToAdd = result.DataToAdd
		op.DataToAddPacked = result.DataToAddPacked
	}

	log.Info("dry run backup complete")
	return nil
}
