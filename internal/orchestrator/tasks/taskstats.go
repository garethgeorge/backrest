package tasks

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/hook"
	"github.com/garethgeorge/backrest/internal/orchestrator"
)

func NewOneoffStatsTask(repoID, planID, at time.Time) orchestrator.Task {
	return &orchestrator.GenericOneoffTask{
		BaseTask: orchestrator.BaseTask{
			TaskName:   fmt.Sprintf("stats for repo %q", repoID),
			TaskRepoID: repoID,
			TaskPlanID: planID,
		},
		OneoffTask: orchestrator.OneoffTask{
			RunAt: at,
			ProtoOp: &v1.Operation{
				Op: &v1.Operation_OperationStats{},
			},
		},
		Do: func(ctx context.Context, st orchestrator.ScheduledTask, taskRunner orchestrator.TaskRunner) error {
			if err := statsHelper(ctx, st, taskRunner); err != nil {
				taskRunner.ExecuteHooks([]v1.Hook_Condition{
					v1.Hook_CONDITION_ANY_ERROR,
				}, hook.HookVars{
					Task:  st.Task.Name(),
					Error: err.Error(),
				})
				return err
			}
			return nil
		},
	}
}

func statsHelper(ctx context.Context, st orchestrator.ScheduledTask, taskRunner orchestrator.TaskRunner) error {
	t := st.Task
	orchestrator := taskRunner.Orchestrator()

	repo, err := orchestrator.GetRepoOrchestrator(t.RepoID())
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
