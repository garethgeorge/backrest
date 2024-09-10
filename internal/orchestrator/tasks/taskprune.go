package tasks

import (
	"context"
	"errors"
	"fmt"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/protoutil"
	"go.uber.org/zap"
)

type PruneTask struct {
	BaseTask
	force  bool
	didRun bool
}

func NewPruneTask(repoID, planID string, force bool) Task {
	return &PruneTask{
		BaseTask: BaseTask{
			TaskType:   "prune",
			TaskName:   fmt.Sprintf("prune repo %q", repoID),
			TaskRepoID: repoID,
			TaskPlanID: planID,
		},
		force: force,
	}
}

func (t *PruneTask) Next(now time.Time, runner TaskRunner) (ScheduledTask, error) {
	if t.force {
		if t.didRun {
			return NeverScheduledTask, nil
		}
		t.didRun = true
		return ScheduledTask{
			Task:  t,
			RunAt: now,
			Op: &v1.Operation{
				Op: &v1.Operation_OperationPrune{},
			},
		}, nil
	}

	repo, err := runner.GetRepo(t.RepoID())
	if err != nil {
		return ScheduledTask{}, fmt.Errorf("get repo %v: %w", t.RepoID(), err)
	}

	if repo.PrunePolicy.GetSchedule() == nil {
		return NeverScheduledTask, nil
	}

	var lastRan time.Time
	var foundBackup bool
	if err := runner.OpLog().Query(oplog.Query{RepoID: t.RepoID(), Reversed: true}, func(op *v1.Operation) error {
		if op.Status == v1.OperationStatus_STATUS_PENDING || op.Status == v1.OperationStatus_STATUS_SYSTEM_CANCELLED {
			return nil
		}
		if _, ok := op.Op.(*v1.Operation_OperationPrune); ok && op.UnixTimeEndMs != 0 {
			lastRan = time.Unix(0, op.UnixTimeEndMs*int64(time.Millisecond))
			return oplog.ErrStopIteration
		}
		if _, ok := op.Op.(*v1.Operation_OperationBackup); ok {
			foundBackup = true
		}
		return nil
	}); err != nil {
		return NeverScheduledTask, fmt.Errorf("finding last prune run time: %w", err)
	} else if !foundBackup {
		lastRan = now
	}

	runAt, err := protoutil.ResolveSchedule(repo.PrunePolicy.GetSchedule(), lastRan, now)
	if errors.Is(err, protoutil.ErrScheduleDisabled) {
		return NeverScheduledTask, nil
	} else if err != nil {
		return NeverScheduledTask, fmt.Errorf("resolve schedule: %w", err)
	}

	return ScheduledTask{
		Task:  t,
		RunAt: runAt,
		Op: &v1.Operation{
			Op: &v1.Operation_OperationPrune{},
		},
	}, nil
}

func (t *PruneTask) Run(ctx context.Context, st ScheduledTask, runner TaskRunner) error {
	op := st.Op

	repo, err := runner.GetRepoOrchestrator(t.RepoID())
	if err != nil {
		return fmt.Errorf("couldn't get repo %q: %w", t.RepoID(), err)
	}

	if err := runner.ExecuteHooks(ctx, []v1.Hook_Condition{
		v1.Hook_CONDITION_PRUNE_START,
	}, HookVars{}); err != nil {
		return fmt.Errorf("prune start hook: %w", err)
	}

	err = repo.UnlockIfAutoEnabled(ctx)
	if err != nil {
		return fmt.Errorf("auto unlock repo %q: %w", t.RepoID(), err)
	}

	opPrune := &v1.Operation_OperationPrune{
		OperationPrune: &v1.OperationPrune{},
	}
	op.Op = opPrune

	liveID, writer, err := runner.LogrefWriter()
	if err != nil {
		return fmt.Errorf("create logref writer: %w", err)
	}
	opPrune.OperationPrune.OutputLogref = liveID

	if err := runner.UpdateOperation(op); err != nil {
		return fmt.Errorf("update operation: %w", err)
	}

	err = repo.Prune(ctx, writer)
	if err != nil {
		runner.ExecuteHooks(ctx, []v1.Hook_Condition{
			v1.Hook_CONDITION_PRUNE_ERROR,
			v1.Hook_CONDITION_ANY_ERROR,
		}, HookVars{
			Error: err.Error(),
		})

		return fmt.Errorf("prune: %w", err)
	}

	frozenID, err := writer.Close()
	if err != nil {
		return fmt.Errorf("close logref writer: %w", err)
	}
	opPrune.OperationPrune.OutputLogref = frozenID

	// Run a stats task after a successful prune
	if err := runner.ScheduleTask(NewStatsTask(t.RepoID(), PlanForSystemTasks, false), TaskPriorityStats); err != nil {
		zap.L().Error("schedule stats task", zap.Error(err))
	}

	if err := runner.ExecuteHooks(ctx, []v1.Hook_Condition{
		v1.Hook_CONDITION_PRUNE_SUCCESS,
	}, HookVars{}); err != nil {
		return fmt.Errorf("execute prune end hooks: %w", err)
	}

	return nil
}
