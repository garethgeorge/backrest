package tasks

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
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
	var foundBackup bool
	if err := runner.OpLog().ForEach(oplog.Query{RepoId: t.RepoID()}, indexutil.Reversed(indexutil.CollectAll()), func(op *v1.Operation) error {
		if _, ok := op.Op.(*v1.Operation_OperationPrune); ok {
			lastRan = time.Unix(0, op.UnixTimeEndMs*int64(time.Millisecond))
			return oplog.ErrStopIteration
		}
		if _, ok := op.Op.(*v1.Operation_OperationBackup); ok {
			foundBackup = true
		}
		return nil
	}); err != nil {
		return NeverScheduledTask, fmt.Errorf("finding last prune run time: %w", err)
	} else if !foundBackup {
		lastRan = time.Now()
	}

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

	if err := runner.ExecuteHooks(ctx, []v1.Hook_Condition{
		v1.Hook_CONDITION_PRUNE_START,
	}, HookVars{}); err != nil {
		return fmt.Errorf("prune start hook: %w", err)
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
	buf := bytes.NewBuffer(nil)
	bufWriter := &ioutil.SynchronizedWriter{W: &ioutil.LimitWriter{W: buf, N: 16 * 1024}}
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

				if opPrune.OperationPrune.Output != string(output) {
					opPrune.OperationPrune.Output = string(output)

					if err := runner.OpLog().Update(op); err != nil {
						zap.L().Error("update prune operation with status output", zap.Error(err))
					}
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	if err := repo.Prune(ctx, bufWriter); err != nil {
		cancel()

		runner.ExecuteHooks(ctx, []v1.Hook_Condition{
			v1.Hook_CONDITION_ANY_ERROR,
		}, HookVars{
			Error: err.Error(),
		})

		return fmt.Errorf("prune: %w", err)
	}
	cancel()
	wg.Wait()

	opPrune.OperationPrune.Output = string(buf.Bytes())

	// Run a stats task after a successful prune
	if err := runner.ScheduleTask(NewStatsTask(t.RepoID(), PlanForSystemTasks, false), TaskPriorityStats); err != nil {
		zap.L().Error("schedule stats task", zap.Error(err))
	}

	if err := runner.ExecuteHooks(ctx, []v1.Hook_Condition{
		v1.Hook_CONDITION_PRUNE_SUCCESS,
	}, HookVars{}); err != nil {
		return fmt.Errorf("execute prune end hooks: %w", err)
	}

	return nil
}
