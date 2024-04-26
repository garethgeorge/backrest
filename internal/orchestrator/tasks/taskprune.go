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
	"github.com/garethgeorge/backrest/internal/orchestrator"
	"go.uber.org/zap"
)

type PruneTask struct {
	orchestrator.BaseTask
	orchestrator.OneoffTask
	force bool
}

func NewOneoffPruneTask(repoID, planID string, flowID int64, at time.Time, force bool) orchestrator.Task {
	return &PruneTask{
		BaseTask: orchestrator.BaseTask{
			TaskName:   fmt.Sprintf("prune for plan %q in repo %q", planID, repoID),
			TaskRepoID: repoID,
			TaskPlanID: planID,
		},
		OneoffTask: orchestrator.OneoffTask{
			FlowID: flowID,
			RunAt:  at,
			ProtoOp: &v1.Operation{
				Op: &v1.Operation_OperationPrune{},
			},
		},
		force: force,
	}
}

func (t *PruneTask) Next(now time.Time, runner orchestrator.TaskRunner) orchestrator.ScheduledTask {
	if t.force {
		return t.OneoffTask.Next(now, runner)
	}

	shouldRun, err := t.shouldRun(now, runner)
	if err != nil {
		zap.S().Errorf("task %v failed to check if it should run: %v", t.Name(), err)
		return orchestrator.NeverScheduledTask
	}
	if !shouldRun {
		return orchestrator.NeverScheduledTask
	}

	return t.OneoffTask.Next(now, runner)
}

func (t *PruneTask) shouldRun(now time.Time, runner orchestrator.TaskRunner) (bool, error) {
	repo, err := runner.Orchestrator().GetRepo(t.RepoID())
	if err != nil {
		return false, fmt.Errorf("get repo %v: %w", t.RepoID(), err)
	}

	nextPruneTime, err := t.getNextPruneTime(runner, repo.PrunePolicy)
	if err != nil {
		return false, fmt.Errorf("get next prune time: %w", err)
	}

	return nextPruneTime.Before(now), nil
}

func (t *PruneTask) getNextPruneTime(runner orchestrator.TaskRunner, policy *v1.PrunePolicy) (time.Time, error) {
	var lastPruneTime time.Time
	runner.OpLog().ForEachByRepo(t.RepoID(), indexutil.Reversed(indexutil.CollectAll()), func(op *v1.Operation) error {
		if _, ok := op.Op.(*v1.Operation_OperationPrune); ok {
			lastPruneTime = time.Unix(0, op.UnixTimeStartMs*int64(time.Millisecond))
			return oplog.ErrStopIteration
		}
		return nil
	})

	if policy != nil {
		return lastPruneTime.Add(time.Duration(policy.MaxFrequencyDays) * 24 * time.Hour), nil
	} else {
		return lastPruneTime.Add(7 * 24 * time.Hour), nil // default to 7 days.
	}
}

func (t *PruneTask) Run(ctx context.Context, st orchestrator.ScheduledTask, runner orchestrator.TaskRunner) error {
	op := st.Op

	repo, err := runner.Orchestrator().GetRepoOrchestrator(t.RepoID())
	if err != nil {
		return fmt.Errorf("couldn't get repo %q: %w", t.RepoID(), err)
	}

	err = repo.UnlockIfAutoEnabled(ctx)
	if err != nil {
		return fmt.Errorf("auto unlock repo %q: %w", t.RepoID(), err)
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

					if err := runner.OpLog().Update(op); err != nil {
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

		runner.ExecuteHooks([]v1.Hook_Condition{
			v1.Hook_CONDITION_ANY_ERROR,
		}, hook.HookVars{
			Error: err.Error(),
		})

		return fmt.Errorf("prune: %w", err)
	}
	cancel()
	wg.Wait()

	// TODO: it would be best to store the output in separate storage for large status data.
	output := buf.String()
	if len(output) > 8*1024 { // only save the first 4K of output.
		output = output[:len(output)-8*1024]
	}

	opPrune.OperationPrune.Output = output

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
