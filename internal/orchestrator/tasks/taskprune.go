package tasks

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/hook"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/oplog/indexutil"
	"go.uber.org/zap"
)

// PruneTask tracks a forget operation.
type PruneTask struct {
	TaskWithOperation
	plan  *v1.Plan
	at    *time.Time
	force bool
}

var _ Task = &PruneTask{}

func NewOneoffPruneTask(orchestrator *Orchestrator, plan *v1.Plan, at time.Time, force bool) *PruneTask {
	return &PruneTask{
		TaskWithOperation: TaskWithOperation{
			orch: orchestrator,
		},
		plan:  plan,
		at:    &at,
		force: force, // overrides the PrunePolicy's MaxFrequencyDays
	}
}

func (t *PruneTask) Name() string {
	return fmt.Sprintf("prune for plan %q", t.plan.Id)
}

func (t *PruneTask) Next(now time.Time) *time.Time {
	shouldRun, err := t.shouldRun(now)
	if err != nil {
		zap.S().Errorf("task %v failed to check if it should run: %v", t.Name(), err)
	}
	if !shouldRun {
		return nil
	}

	ret := t.at
	if ret != nil {
		t.at = nil
		if err := t.setOperation(&v1.Operation{
			PlanId:          t.plan.Id,
			RepoId:          t.plan.Repo,
			UnixTimeStartMs: timeToUnixMillis(*ret),
			Status:          v1.OperationStatus_STATUS_PENDING,
			Op:              &v1.Operation_OperationPrune{},
		}); err != nil {
			zap.S().Errorf("task %v failed to add operation to oplog: %v", t.Name(), err)
			return nil
		}
	}
	return ret
}

func (t *PruneTask) shouldRun(now time.Time) (bool, error) {
	if t.force {
		return true, nil
	}

	repo, err := t.orch.GetRepo(t.plan.Repo)
	if err != nil {
		return false, fmt.Errorf("get repo %v: %w", t.plan.Repo, err)
	}

	nextPruneTime, err := t.getNextPruneTime(repo, repo.repoConfig.PrunePolicy)
	if err != nil {
		return false, fmt.Errorf("get next prune time: %w", err)
	}

	return nextPruneTime.Before(now), nil
}

func (t *PruneTask) getNextPruneTime(repo *RepoOrchestrator, policy *v1.PrunePolicy) (time.Time, error) {
	var lastPruneTime time.Time
	t.orch.OpLog.ForEachByRepo(t.plan.Repo, indexutil.Reversed(indexutil.CollectAll()), func(op *v1.Operation) error {
		if _, ok := op.Op.(*v1.Operation_OperationPrune); ok {
			lastPruneTime = time.Unix(0, op.UnixTimeStartMs*int64(time.Millisecond))
			return oplog.ErrStopIteration
		}
		return nil
	})

	if repo.repoConfig.PrunePolicy != nil {
		return lastPruneTime.Add(time.Duration(repo.repoConfig.PrunePolicy.MaxFrequencyDays) * 24 * time.Hour), nil
	} else {
		return lastPruneTime.Add(7 * 24 * time.Hour), nil // default to 7 days.
	}
}

func (t *PruneTask) Run(ctx context.Context) error {
	if err := t.runWithOpAndContext(ctx, func(ctx context.Context, op *v1.Operation) error {
		repo, err := t.orch.GetRepo(t.plan.Repo)
		if err != nil {
			return fmt.Errorf("get repo %v: %w", t.plan.Repo, err)
		}

		err = repo.UnlockIfAutoEnabled(ctx)
		if err != nil {
			return fmt.Errorf("auto unlock repo %q: %w", t.plan.Repo, err)
		}

		opPrune := &v1.Operation_OperationPrune{
			OperationPrune: &v1.OperationPrune{},
		}
		op.Op = opPrune

		ctx, cancel := context.WithCancel(ctx)
		interval := time.NewTicker(1 * time.Second)
		defer interval.Stop()
		var buf synchronizedBuffer
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {

			defer wg.Done()
			for {
				select {
				case <-interval.C:
					output := buf.String()
					if len(output) > 8*1024 { // only provide live status upto the first 8K of output.
						output = output[:len(output)-8*1024]
					}

					if opPrune.OperationPrune.Output != output {
						opPrune.OperationPrune.Output = buf.String()

						if err := t.orch.OpLog.Update(op); err != nil {
							zap.L().Error("update prune operation with status output", zap.Error(err))
						}
					}
				case <-ctx.Done():
					return
				}
			}
		}()

		if err := repo.Prune(ctx, &buf); err != nil {
			cancel()
			return fmt.Errorf("prune: %w", err)
		}
		cancel()
		wg.Wait()

		// TODO: it would be best to store the output in separate storage for large status data.
		output := buf.String()
		if len(output) > 8*1024 { // only save the first 4K of output.
			output = output[:len(output)-8*1024]
		}
		op.Op = &v1.Operation_OperationPrune{
			OperationPrune: &v1.OperationPrune{
				Output: output,
			},
		}

		return nil
	}); err != nil {
		repo, _ := t.orch.GetRepo(t.plan.Repo)
		_ = t.orch.hookExecutor.ExecuteHooks(repo.Config(), t.plan, []v1.Hook_Condition{
			v1.Hook_CONDITION_ANY_ERROR,
		}, hook.HookVars{
			Task:  t.Name(),
			Error: err.Error(),
		})
		return err
	}

	t.orch.ScheduleTask(NewOneoffStatsTask(t.orch, t.plan.Repo, t.plan.Id, time.Now()), TaskPriorityStats)

	return nil
}

// synchronizedBuffer is used for collecting prune command's output
type synchronizedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (w *synchronizedBuffer) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.buf.Write(p)
}

func (w *synchronizedBuffer) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.buf.String()
}
