package tasks

import (
	"context"
	"errors"
	"fmt"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/protoutil"
	"github.com/hashicorp/go-multierror"
	"go.uber.org/zap"
)

type ScheduledForgetTask struct {
	BaseTask
	force  bool
	didRun bool
}

func NewScheduledForgetTask(repo *v1.Repo, planID string, force bool) Task {
	return &ScheduledForgetTask{
		BaseTask: BaseTask{
			TaskType:   "scheduled_forget",
			TaskName:   fmt.Sprintf("scheduled forget for repo %q", repo.Id),
			TaskRepo:   repo,
			TaskPlanID: planID,
		},
		force: force,
	}
}

func (t *ScheduledForgetTask) Next(now time.Time, runner TaskRunner) (ScheduledTask, error) {
	if t.force {
		if t.didRun {
			return NeverScheduledTask, nil
		}
		t.didRun = true
		return ScheduledTask{
			Task:  t,
			RunAt: now,
			Op: &v1.Operation{
				Op: &v1.Operation_OperationForget{},
			},
		}, nil
	}

	repo, err := runner.GetRepo(t.RepoID())
	if err != nil {
		return ScheduledTask{}, fmt.Errorf("get repo %v: %w", t.RepoID(), err)
	}

	if repo.GetForgetPolicy().GetSchedule() == nil {
		return NeverScheduledTask, nil
	}

	var lastRan time.Time
	var foundBackup bool
	if err := runner.QueryOperations(oplog.Query{}.
		SetInstanceID(runner.InstanceID()).
		SetRepoGUID(repo.GetGuid()).
		SetPlanID(PlanForSystemTasks).
		SetReversed(true), func(op *v1.Operation) error {
		if op.Status == v1.OperationStatus_STATUS_PENDING || op.Status == v1.OperationStatus_STATUS_SYSTEM_CANCELLED {
			return nil
		}
		if _, ok := op.Op.(*v1.Operation_OperationForget); ok && op.UnixTimeEndMs != 0 {
			lastRan = time.Unix(0, op.UnixTimeEndMs*int64(time.Millisecond))
			return oplog.ErrStopIteration
		}
		if _, ok := op.Op.(*v1.Operation_OperationBackup); ok {
			foundBackup = true
		}
		return nil
	}); err != nil {
		return NeverScheduledTask, fmt.Errorf("finding last scheduled forget run time: %w", err)
	} else if !foundBackup {
		lastRan = now
	}

	runAt, err := protoutil.ResolveSchedule(repo.GetForgetPolicy().GetSchedule(), lastRan, now)
	if errors.Is(err, protoutil.ErrScheduleDisabled) {
		return NeverScheduledTask, nil
	} else if err != nil {
		return NeverScheduledTask, fmt.Errorf("resolve schedule: %w", err)
	}

	return ScheduledTask{
		Task:  t,
		RunAt: runAt,
		Op: &v1.Operation{
			Op: &v1.Operation_OperationForget{},
		},
	}, nil
}

// shouldSkip returns true if there are no new successful backups since the last scheduled forget.
func (t *ScheduledForgetTask) shouldSkip(runner TaskRunner, repo *v1.Repo) bool {
	var lastForgetEndMs int64
	var hasNewBackup bool

	_ = runner.QueryOperations(oplog.Query{}.
		SetInstanceID(runner.InstanceID()).
		SetRepoGUID(repo.GetGuid()).
		SetPlanID(PlanForSystemTasks).
		SetReversed(true), func(op *v1.Operation) error {
		if op.Status != v1.OperationStatus_STATUS_SUCCESS {
			return nil
		}
		if _, ok := op.Op.(*v1.Operation_OperationForget); ok && op.UnixTimeEndMs != 0 {
			lastForgetEndMs = op.UnixTimeEndMs
			return oplog.ErrStopIteration
		}
		return nil
	})

	if lastForgetEndMs == 0 {
		return false // no previous forget, don't skip
	}

	// Check if any backup completed after the last forget
	_ = runner.QueryOperations(oplog.Query{}.
		SetRepoGUID(repo.GetGuid()).
		SetReversed(true), func(op *v1.Operation) error {
		if op.UnixTimeEndMs < lastForgetEndMs {
			return oplog.ErrStopIteration // older than last forget, stop looking
		}
		if op.Status == v1.OperationStatus_STATUS_SUCCESS {
			if _, ok := op.Op.(*v1.Operation_OperationBackup); ok {
				hasNewBackup = true
				return oplog.ErrStopIteration
			}
		}
		return nil
	})

	return !hasNewBackup
}

func (t *ScheduledForgetTask) Run(ctx context.Context, st ScheduledTask, runner TaskRunner) error {
	op := st.Op

	notifyError := func(err error) error {
		return NotifyError(ctx, runner, t.Name(), err, v1.Hook_CONDITION_FORGET_ERROR)
	}

	repo, err := runner.GetRepo(t.RepoID())
	if err != nil {
		return notifyError(fmt.Errorf("get repo %q: %w", t.RepoID(), err))
	}

	// Skip if no new backups since last forget run.
	// Mark as system-cancelled so it doesn't count as a successful run
	// and the next schedule is computed from the last actual forget.
	if t.shouldSkip(runner, repo) {
		op.Op = &v1.Operation_OperationForget{
			OperationForget: &v1.OperationForget{},
		}
		op.Status = v1.OperationStatus_STATUS_SYSTEM_CANCELLED
		op.DisplayMessage = "Skipped: no new backups since last forget"
		return nil
	}

	r, err := runner.GetRepoOrchestrator(t.RepoID())
	if err != nil {
		return notifyError(fmt.Errorf("get repo orchestrator %q: %w", t.RepoID(), err))
	}

	if err := r.UnlockIfAutoEnabled(ctx); err != nil {
		return notifyError(fmt.Errorf("auto unlock repo %q: %w", t.RepoID(), err))
	}

	if err := runner.ExecuteHooks(ctx, []v1.Hook_Condition{
		v1.Hook_CONDITION_FORGET_START,
	}, HookVars{}); err != nil {
		return notifyError(fmt.Errorf("forget start hook: %w", err))
	}

	forgot, err := r.ForgetAll(ctx, repo.GetForgetPolicy().GetRetention())

	forgetOp := &v1.Operation_OperationForget{
		OperationForget: &v1.OperationForget{
			Forget: forgot,
			Policy: repo.GetForgetPolicy().GetRetention(),
		},
	}
	op.Op = forgetOp

	// Mark forgotten snapshots in the oplog
	var ops []*v1.Operation
	for _, f := range forgot {
		if e := runner.QueryOperations(oplog.Query{}.
			SetRepoGUID(t.Repo().GetGuid()).
			SetSnapshotID(f.Id), func(op *v1.Operation) error {
			ops = append(ops, op)
			return nil
		}); e != nil {
			err = multierror.Append(err, fmt.Errorf("lookup snapshot %v: %w", f.Id, e))
		}
	}

	l := runner.Logger(ctx)
	l.Sugar().Debugf("found %v snapshots were forgotten, marking this in oplog", len(ops))

	for _, op := range ops {
		if indexOp, ok := op.Op.(*v1.Operation_OperationIndexSnapshot); ok {
			indexOp.OperationIndexSnapshot.Forgot = true
			if e := runner.UpdateOperation(op); e != nil {
				err = multierror.Append(err, fmt.Errorf("mark index snapshot %v as forgotten: %w", op.Id, e))
			}
		}
	}

	if err != nil {
		return notifyError(fmt.Errorf("scheduled forget: %w", err))
	}

	// Schedule a stats task after successful forget
	if e := runner.ScheduleTask(NewStatsTask(t.Repo(), PlanForSystemTasks, false), TaskPriorityStats); e != nil {
		zap.L().Error("schedule stats task", zap.Error(e))
	}

	if e := runner.ExecuteHooks(ctx, []v1.Hook_Condition{
		v1.Hook_CONDITION_FORGET_SUCCESS,
	}, HookVars{}); e != nil {
		return fmt.Errorf("forget end hook: %w", e)
	}

	return nil
}
