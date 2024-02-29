package orchestrator

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/hook"
	"go.uber.org/zap"
)

// StatsTask tracks a restic stats operation.
type StatsTask struct {
	TaskWithOperation
	planId string
	repoId string
	at     *time.Time
}

var _ Task = &StatsTask{}

func NewOneoffStatsTask(orchestrator *Orchestrator, repoId, planId string, at time.Time) *StatsTask {
	return &StatsTask{
		TaskWithOperation: TaskWithOperation{
			orch: orchestrator,
		},
		planId: planId,
		repoId: repoId,
		at:     &at,
	}
}

func (t *StatsTask) Name() string {
	return fmt.Sprintf("stats for repo %q", t.repoId)
}

func (t *StatsTask) Next(now time.Time) *time.Time {
	ret := t.at
	if ret != nil {
		t.at = nil

		if err := t.setOperation(&v1.Operation{
			PlanId:          t.planId,
			RepoId:          t.repoId,
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
	if err := t.runWithOpAndContext(ctx, func(ctx context.Context, op *v1.Operation) error {
		repo, err := t.orch.GetRepo(t.repoId)
		if err != nil {
			return fmt.Errorf("get repo %q: %w", t.repoId, err)
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
		repo, _ := t.orch.GetRepo(t.repoId)
		plan, _ := t.orch.GetPlan(t.planId)
		t.orch.hookExecutor.ExecuteHooks(repo.Config(), plan, "", []v1.Hook_Condition{
			v1.Hook_CONDITION_ANY_ERROR,
		}, hook.HookVars{
			Task:  t.Name(),
			Error: err.Error(),
		})
		return err
	}
	return nil
}
