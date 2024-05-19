package tasks

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/hook"
	"github.com/garethgeorge/backrest/internal/ioutil"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/oplog/indexutil"
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
	if err := runner.OpLog().ForEach(oplog.Query{RepoId: t.RepoID()}, indexutil.Reversed(indexutil.CollectAll()), func(op *v1.Operation) error {
		if _, ok := op.Op.(*v1.Operation_OperationPrune); ok {
			lastRan = time.Unix(0, op.UnixTimeEndMs*int64(time.Millisecond))
			return oplog.ErrStopIteration
		}
		return nil
	}); err != nil {
		return NeverScheduledTask, fmt.Errorf("finding last backup run time: %w", err)
	}

	zap.L().Debug("last prune time", zap.Time("time", lastRan), zap.String("repo", t.RepoID()))

	runAt, err := protoutil.ResolveSchedule(repo.PrunePolicy.GetSchedule(), lastRan)
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
	var buf bytes.Buffer
	bufWriter := ioutil.SynchronizedWriter{W: &buf}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-interval.C:
				bufWriter.Mu.Lock()
				output := buf.String()
				bufWriter.Mu.Unlock()
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

	if err := repo.Prune(ctx, &bufWriter); err != nil {
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

	output := buf.String()
	if len(output) > 8*1024 { // only save the first 4K of output.
		output = output[:len(output)-8*1024]
	}

	opPrune.OperationPrune.Output = output

	return nil
}
