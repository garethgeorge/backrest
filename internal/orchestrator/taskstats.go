package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/hook"
	"go.uber.org/zap"
)

// StatsTask tracks a restic stats operation.
type StatsTask struct {
	TaskWithOperation
	plan *v1.Plan
	at   *time.Time
}

var _ Task = &StatsTask{}

func NewOneoffStatsTask(orchestrator *Orchestrator, plan *v1.Plan, at time.Time) *StatsTask {
	return &StatsTask{
		TaskWithOperation: TaskWithOperation{
			orch: orchestrator,
		},
		plan: plan,
		at:   &at,
	}
}

func (t *StatsTask) Name() string {
	return fmt.Sprintf("stats for plan %q", t.plan.Id)
}

func (t *StatsTask) Next(now time.Time) *time.Time {
	ret := t.at
	if ret != nil {
		t.at = nil

		if err := t.setOperation(&v1.Operation{
			PlanId:          t.plan.Id,
			RepoId:          t.plan.Repo,
			UnixTimeStartMs: timeToUnixMillis(*ret),
			Status:          v1.OperationStatus_STATUS_PENDING,
			Op:              &v1.Operation_OperationStats{},
		}); err != nil {
			zap.S().Errorf("task %v failed to add operation to oplog: %v", t.Name(), err)
			return nil
		}
	}
	return ret
}

func (t *StatsTask) Run(ctx context.Context) error {
	if t.plan.Retention == nil {
		return errors.New("plan does not have a retention policy")
	}

	if err := t.runWithOpAndContext(ctx, func(ctx context.Context, op *v1.Operation) error {
		repo, err := t.orch.GetRepo(t.plan.Repo)
		if err != nil {
			return fmt.Errorf("get repo %q: %w", t.plan.Repo, err)
		}

		stats, err := repo.Stats(ctx)
		if err != nil {
			return fmt.Errorf("get stats: %w", err)
		}

		op.Op = &v1.Operation_OperationStats{
			OperationStats: &v1.OperationStats{
				Stats: stats,
			},
		}

		return err
	}); err != nil {
		repo, _ := t.orch.GetRepo(t.plan.Repo)
		t.orch.hookExecutor.ExecuteHooks(repo.Config(), t.plan, "", []v1.Hook_Condition{
			v1.Hook_CONDITION_ANY_ERROR,
		}, hook.HookVars{
			Task:  t.Name(),
			Error: err.Error(),
		})
		return err
	}
	return nil
}
