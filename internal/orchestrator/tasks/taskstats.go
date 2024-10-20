package tasks

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/oplog"
)

type StatsTask struct {
	BaseTask
	force  bool
	didRun bool
}

func NewStatsTask(repoID, planID string, force bool) Task {
	return &StatsTask{
		BaseTask: BaseTask{
			TaskType:   "stats",
			TaskName:   fmt.Sprintf("stats for repo %q", repoID),
			TaskRepoID: repoID,
			TaskPlanID: planID,
		},
		force: force,
	}
}

func (t *StatsTask) Next(now time.Time, runner TaskRunner) (ScheduledTask, error) {
	if t.force {
		if t.didRun {
			return NeverScheduledTask, nil
		}
		t.didRun = true
		return ScheduledTask{
			Task:  t,
			RunAt: now,
			Op: &v1.Operation{
				Op: &v1.Operation_OperationStats{},
			},
		}, nil
	}

	// check last stats time
	var lastRan time.Time
	if err := runner.OpLog().Query(oplog.Query{RepoID: t.RepoID(), Reversed: true}, func(op *v1.Operation) error {
		if op.Status == v1.OperationStatus_STATUS_PENDING || op.Status == v1.OperationStatus_STATUS_SYSTEM_CANCELLED {
			return nil
		}
		if _, ok := op.Op.(*v1.Operation_OperationStats); ok && op.UnixTimeEndMs != 0 {
			lastRan = time.Unix(0, op.UnixTimeEndMs*int64(time.Millisecond))
			return oplog.ErrStopIteration
		}
		return nil
	}); err != nil {
		return NeverScheduledTask, fmt.Errorf("finding last check run time: %w", err)
	}

	// Runs at most once per day.
	if time.Since(lastRan) < 24*time.Hour {
		return NeverScheduledTask, nil
	}
	return ScheduledTask{
		Task:  t,
		RunAt: now,
		Op: &v1.Operation{
			Op: &v1.Operation_OperationStats{},
		},
	}, nil
}

func (t *StatsTask) Run(ctx context.Context, st ScheduledTask, runner TaskRunner) error {
	if err := statsHelper(ctx, st, runner); err != nil {
		runner.ExecuteHooks(ctx, []v1.Hook_Condition{
			v1.Hook_CONDITION_ANY_ERROR,
		}, HookVars{
			Task:  st.Task.Name(),
			Error: err.Error(),
		})
		return err
	}

	return nil
}

func statsHelper(ctx context.Context, st ScheduledTask, taskRunner TaskRunner) error {
	t := st.Task

	repo, err := taskRunner.GetRepoOrchestrator(t.RepoID())
	if err != nil {
		return fmt.Errorf("get repo %q: %w", t.RepoID(), err)
	}

	stats, err := repo.Stats(ctx)
	if err != nil {
		return fmt.Errorf("get stats: %w", err)
	}

	op := st.Op
	op.Op = &v1.Operation_OperationStats{
		OperationStats: &v1.OperationStats{
			Stats: stats,
		},
	}

	return nil
}
