package tasks

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/orchestrator/repo"
	"github.com/garethgeorge/backrest/internal/protoutil"
	"github.com/garethgeorge/backrest/pkg/restic"
	"github.com/hashicorp/go-multierror"
	"go.uber.org/zap"
)

// NewOneoffForgetTask creates a per-plan forget task that runs once after a backup.
// It applies the plan's retention policy scoped to snapshots tagged for that plan.
func NewOneoffForgetTask(repoProto *v1.Repo, planID string, flowID int64, at time.Time) Task {
	return &GenericOneoffTask{
		BaseTask: BaseTask{
			TaskType:   "forget",
			TaskName:   fmt.Sprintf("forget for plan %q in repo %q", planID, repoProto.Id),
			TaskRepo:   repoProto,
			TaskPlanID: planID,
		},
		FlowID: flowID,
		RunAt:  at,
		ProtoOp: &v1.Operation{
			Op: &v1.Operation_OperationForget{},
		},
		Do: func(ctx context.Context, st ScheduledTask, runner TaskRunner) error {
			if st.Op.GetOperationForget() == nil {
				panic("forget task with non-forget operation")
			}

			t := st.Task
			l := runner.Logger(ctx)

			plan, err := runner.GetPlan(t.PlanID())
			if err != nil {
				return fmt.Errorf("get plan %q: %w", t.PlanID(), err)
			}

			tags := []string{repo.TagForPlan(t.PlanID())}
			if compat, err := UseLegacyCompatMode(l, runner, t.Repo().GetGuid(), t.PlanID()); err != nil {
				return fmt.Errorf("check legacy compat mode: %w", err)
			} else if !compat {
				tags = append(tags, repo.TagForInstance(runner.Config().Instance))
			} else {
				l.Warn("forgetting snapshots without instance ID, using legacy behavior (e.g. --tags not including instance ID)")
				l.Sugar().Warnf("to avoid this warning, tag all snapshots with the instance ID e.g. by running: \r\n"+
					"restic tag --set '%s' --set '%s' --tag '%s'", repo.TagForPlan(t.PlanID()), repo.TagForInstance(runner.Config().Instance), repo.TagForPlan(t.PlanID()))
			}

			return forgetHelper(ctx, st, runner, plan.Retention,
				restic.WithFlags("--tag", strings.Join(tags, ",")),
				restic.WithFlags("--group-by", ""),
			)
		},
	}
}

// ScheduledForgetTask is a repo-level forget task that runs on a schedule.
// It applies the repo's forget policy retention to all snapshots, grouped by tags.
type ScheduledForgetTask struct {
	BaseTask
	force  bool
	didRun bool
}

func NewScheduledForgetTask(repoProto *v1.Repo, planID string, force bool) Task {
	return &ScheduledForgetTask{
		BaseTask: BaseTask{
			TaskType:   "scheduled_forget",
			TaskName:   fmt.Sprintf("scheduled forget for repo %q", repoProto.Id),
			TaskRepo:   repoProto,
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
			RunAt: now,
			Op: &v1.Operation{
				Op: &v1.Operation_OperationForget{},
			},
		}, nil
	}

	repoProto, err := runner.GetRepo(t.RepoID())
	if err != nil {
		return ScheduledTask{}, fmt.Errorf("get repo %v: %w", t.RepoID(), err)
	}

	if repoProto.GetForgetPolicy().GetSchedule() == nil {
		return NeverScheduledTask, nil
	}

	var lastRan time.Time
	var foundBackup bool
	if err := runner.QueryOperations(oplog.Query{}.
		SetRepoGUID(repoProto.GetGuid()).
		SetReversed(true), func(op *v1.Operation) error {
		if op.Status == v1.OperationStatus_STATUS_PENDING {
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

	runAt, err := protoutil.ResolveSchedule(repoProto.GetForgetPolicy().GetSchedule(), lastRan, now)
	if errors.Is(err, protoutil.ErrScheduleDisabled) {
		return NeverScheduledTask, nil
	} else if err != nil {
		return NeverScheduledTask, fmt.Errorf("resolve schedule: %w", err)
	}

	return ScheduledTask{
		RunAt: runAt,
		Op: &v1.Operation{
			Op: &v1.Operation_OperationForget{},
		},
	}, nil
}

// shouldSkip returns true if there are no new successful backups since the last scheduled forget.
func (t *ScheduledForgetTask) shouldSkip(runner TaskRunner, repoProto *v1.Repo) bool {
	var lastForgetEndMs int64
	var hasNewBackup bool

	_ = runner.QueryOperations(oplog.Query{}.
		SetInstanceID(runner.InstanceID()).
		SetRepoGUID(repoProto.GetGuid()).
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

	// Check if any backup completed after the last forget.
	// Intentionally not scoped by instance ID: in a sync setup the server receives
	// backup operations from remote clients. We want forget to run whenever new
	// snapshots appear in the repo regardless of which instance created them.
	_ = runner.QueryOperations(oplog.Query{}.
		SetRepoGUID(repoProto.GetGuid()).
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

	repoProto, err := runner.GetRepo(t.RepoID())
	if err != nil {
		return NotifyError(ctx, runner, t.Name(), fmt.Errorf("get repo %q: %w", t.RepoID(), err), v1.Hook_CONDITION_FORGET_ERROR)
	}

	// Skip if no new backups since last forget run.
	// Mark as system-cancelled so it doesn't count as a successful run
	// and the next schedule is computed from the last actual forget.
	// Interactive (force) runs always execute.
	if !t.force && t.shouldSkip(runner, repoProto) {
		op.Op = &v1.Operation_OperationForget{
			OperationForget: &v1.OperationForget{},
		}
		op.Status = v1.OperationStatus_STATUS_SYSTEM_CANCELLED
		op.DisplayMessage = "Skipped: no new backups since last forget"
		return nil
	}

	err = forgetHelper(ctx, st, runner, repoProto.GetForgetPolicy().GetRetention(),
		restic.WithFlags("--group-by", "tags"),
	)
	if err != nil {
		return err
	}

	// Schedule a stats task after successful forget
	if e := runner.ScheduleTask(NewStatsTask(t.Repo(), PlanForSystemTasks, false), TaskPriorityStats); e != nil {
		zap.L().Error("schedule stats task", zap.Error(e))
	}

	return nil
}

// forgetHelper contains the shared logic for running a forget operation.
// It handles unlock, hooks, calling restic forget, and marking forgotten snapshots in the oplog.
func forgetHelper(ctx context.Context, st ScheduledTask, runner TaskRunner, policy *v1.RetentionPolicy, opts ...restic.GenericOption) error {
	t := st.Task

	notifyError := func(err error) error {
		return NotifyError(ctx, runner, t.Name(), err, v1.Hook_CONDITION_FORGET_ERROR)
	}

	r, err := runner.GetRepoOrchestrator(t.RepoID())
	if err != nil {
		return notifyError(fmt.Errorf("get repo %q: %w", t.RepoID(), err))
	}

	if err := r.UnlockIfAutoEnabled(ctx); err != nil {
		return notifyError(fmt.Errorf("auto unlock repo %q: %w", t.RepoID(), err))
	}

	if err := runner.ExecuteHooks(ctx, []v1.Hook_Condition{
		v1.Hook_CONDITION_FORGET_START,
	}, HookVars{}); err != nil {
		return notifyError(fmt.Errorf("forget start hook: %w", err))
	}

	forgot, err := r.Forget(ctx, policy, opts...)

	forgetOp := &v1.Operation_OperationForget{
		OperationForget: &v1.OperationForget{
			Forget: forgot,
			Policy: policy,
		},
	}
	st.Op.Op = forgetOp

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
		return notifyError(fmt.Errorf("forget: %w", err))
	}

	if e := runner.ExecuteHooks(ctx, []v1.Hook_Condition{
		v1.Hook_CONDITION_FORGET_SUCCESS,
	}, HookVars{}); e != nil {
		return fmt.Errorf("forget end hook: %w", e)
	}

	return nil
}

// UseLegacyCompatMode checks if there are any snapshots that were created without a `created-by` tag still exist in the repo.
// The property is overridden if mixed `created-by` tag values are found.
func UseLegacyCompatMode(l *zap.Logger, taskRunner TaskRunner, repoGUID, planID string) (bool, error) {
	instanceIDs := make(map[string]struct{})
	if err := taskRunner.QueryOperations(oplog.Query{}.SetRepoGUID(repoGUID).SetPlanID(planID).SetReversed(true), func(op *v1.Operation) error {
		if snapshotOp, ok := op.Op.(*v1.Operation_OperationIndexSnapshot); ok && !snapshotOp.OperationIndexSnapshot.GetForgot() {
			tags := snapshotOp.OperationIndexSnapshot.GetSnapshot().GetTags()
			instanceIDs[repo.InstanceIDFromTags(tags)] = struct{}{}
		}
		return nil
	}); err != nil {
		return false, err
	}
	if _, ok := instanceIDs[""]; !ok {
		return false, nil
	}
	delete(instanceIDs, "")
	if len(instanceIDs) > 1 {
		l.Sugar().Warn("found mixed instance IDs in indexed snapshots, overriding legacy forget behavior to include instance ID tags. This may result in unexpected behavior -- please inspect the tags on your snapshots.")
		return false, nil
	}
	l.Sugar().Warn("found legacy snapshots without instance ID, recommending legacy forget behavior.")
	return true, nil
}
