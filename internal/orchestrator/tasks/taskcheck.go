package tasks

import (
	"context"
	"errors"
	"fmt"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/protoutil"
)

type CheckTask struct {
	BaseTask
	force  bool
	didRun bool
}

func NewCheckTask(repo *v1.Repo, planID string, force bool) Task {
	return &CheckTask{
		BaseTask: BaseTask{
			TaskType:   "check",
			TaskName:   fmt.Sprintf("check for repo %q", repo.Id),
			TaskRepo:   repo,
			TaskPlanID: planID,
		},
		force: force,
	}
}

func (t *CheckTask) Next(now time.Time, runner TaskRunner) (ScheduledTask, error) {
	if t.force {
		if t.didRun {
			return NeverScheduledTask, nil
		}
		t.didRun = true
		return ScheduledTask{
			Task:  t,
			RunAt: now,
			Op: &v1.Operation{
				Op: &v1.Operation_OperationCheck{},
			},
		}, nil
	}

	repo, err := runner.GetRepo(t.RepoID())
	if err != nil {
		return ScheduledTask{}, fmt.Errorf("get repo %v: %w", t.RepoID(), err)
	}

	if repo.CheckPolicy.GetSchedule() == nil {
		return NeverScheduledTask, nil
	}

	var lastRan time.Time
	var foundBackup bool

	if err := runner.QueryOperations(oplog.Query{}.
		SetInstanceID(runner.InstanceID()). // note: this means that check tasks run by remote instances are ignored.
		SetRepoGUID(t.Repo().GetGuid()).
		SetReversed(true), func(op *v1.Operation) error {
		if op.Status == v1.OperationStatus_STATUS_PENDING || op.Status == v1.OperationStatus_STATUS_SYSTEM_CANCELLED {
			return nil
		}
		if _, ok := op.Op.(*v1.Operation_OperationCheck); ok && op.UnixTimeEndMs != 0 {
			lastRan = time.Unix(0, op.UnixTimeEndMs*int64(time.Millisecond))
			return oplog.ErrStopIteration
		}
		if _, ok := op.Op.(*v1.Operation_OperationBackup); ok {
			foundBackup = true
		}
		return nil
	}); err != nil {
		return NeverScheduledTask, fmt.Errorf("finding last check run time: %w", err)
	} else if !foundBackup {
		lastRan = now
	}

	runAt, err := protoutil.ResolveSchedule(repo.CheckPolicy.GetSchedule(), lastRan, now)
	if errors.Is(err, protoutil.ErrScheduleDisabled) {
		return NeverScheduledTask, nil
	} else if err != nil {
		return NeverScheduledTask, fmt.Errorf("resolve schedule: %w", err)
	}

	return ScheduledTask{
		Task:  t,
		RunAt: runAt,
		Op: &v1.Operation{
			Op: &v1.Operation_OperationCheck{},
		},
	}, nil
}

func (t *CheckTask) Run(ctx context.Context, st ScheduledTask, runner TaskRunner) error {
	op := st.Op

	// Helper to notify of errors during the setup phase
	notifyError := func(err error) error {
		return NotifyError(ctx, runner, t.Name(), err, v1.Hook_CONDITION_CHECK_ERROR)
	}

	repo, err := runner.GetRepoOrchestrator(t.RepoID())
	if err != nil {
		return notifyError(fmt.Errorf("couldn't get repo %q: %w", t.RepoID(), err))
	}

	if err := runner.ExecuteHooks(ctx, []v1.Hook_Condition{
		v1.Hook_CONDITION_CHECK_START,
	}, HookVars{}); err != nil {
		return notifyError(fmt.Errorf("check start hook: %w", err))
	}

	err = repo.UnlockIfAutoEnabled(ctx)
	if err != nil {
		return notifyError(fmt.Errorf("auto unlock repo %q: %w", t.RepoID(), err))
	}

	opCheck := &v1.Operation_OperationCheck{
		OperationCheck: &v1.OperationCheck{},
	}
	op.Op = opCheck

	liveID, writer, err := runner.LogrefWriter()
	if err != nil {
		return fmt.Errorf("create logref writer: %w", err)
	}
	defer writer.Close()
	opCheck.OperationCheck.OutputLogref = liveID

	if err := runner.UpdateOperation(op); err != nil {
		return fmt.Errorf("update operation: %w", err)
	}

	err = repo.Check(ctx, writer)
	if err != nil {
		runner.ExecuteHooks(ctx, []v1.Hook_Condition{
			v1.Hook_CONDITION_CHECK_ERROR,
			v1.Hook_CONDITION_ANY_ERROR,
		}, HookVars{
			Error: err.Error(),
		})

		return fmt.Errorf("check: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("close logref writer: %w", err)
	}

	if err := runner.ExecuteHooks(ctx, []v1.Hook_Condition{
		v1.Hook_CONDITION_CHECK_SUCCESS,
	}, HookVars{}); err != nil {
		return fmt.Errorf("execute check success hooks: %w", err)
	}

	return nil
}
