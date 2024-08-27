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
	"github.com/garethgeorge/backrest/internal/protoutil"
	"go.uber.org/zap"
)

type CheckTask struct {
	BaseTask
	force  bool
	didRun bool
}

func NewCheckTask(repoID, planID string, force bool) Task {
	return &CheckTask{
		BaseTask: BaseTask{
			TaskName:   fmt.Sprintf("check for repo %q", repoID),
			TaskRepoID: repoID,
			TaskPlanID: planID,
		},
		force: force,
	}
}

func (t *CheckTask) Next(now time.Time, runner TaskRunner) (ScheduledTask, error) {
	if t.force {
		if t.didRun {
			return NeverScheduledTask, nil
		}
		t.didRun = true
		return ScheduledTask{
			Task:  t,
			RunAt: now,
			Op: &v1.Operation{
				Op: &v1.Operation_OperationCheck{},
			},
		}, nil
	}

	repo, err := runner.GetRepo(t.RepoID())
	if err != nil {
		return ScheduledTask{}, fmt.Errorf("get repo %v: %w", t.RepoID(), err)
	}

	if repo.CheckPolicy.GetSchedule() == nil {
		return NeverScheduledTask, nil
	}

	var lastRan time.Time
	var foundBackup bool

	if err := runner.OpLog().Query(oplog.Query{RepoID: t.RepoID(), Reversed: true}, func(op *v1.Operation) error {
		if _, ok := op.Op.(*v1.Operation_OperationCheck); ok {
			lastRan = time.Unix(0, op.UnixTimeEndMs*int64(time.Millisecond))
			return oplog.ErrStopIteration
		}
		if _, ok := op.Op.(*v1.Operation_OperationBackup); ok {
			foundBackup = true
		}
		return nil
	}); err != nil {
		return NeverScheduledTask, fmt.Errorf("finding last check run time: %w", err)
	} else if !foundBackup {
		lastRan = now
	}

	runAt, err := protoutil.ResolveSchedule(repo.CheckPolicy.GetSchedule(), lastRan, now)
	if errors.Is(err, protoutil.ErrScheduleDisabled) {
		return NeverScheduledTask, nil
	} else if err != nil {
		return NeverScheduledTask, fmt.Errorf("resolve schedule: %w", err)
	}

	return ScheduledTask{
		Task:  t,
		RunAt: runAt,
		Op: &v1.Operation{
			Op: &v1.Operation_OperationCheck{},
		},
	}, nil
}

func (t *CheckTask) Run(ctx context.Context, st ScheduledTask, runner TaskRunner) error {
	op := st.Op

	repo, err := runner.GetRepoOrchestrator(t.RepoID())
	if err != nil {
		return fmt.Errorf("couldn't get repo %q: %w", t.RepoID(), err)
	}

	if err := runner.ExecuteHooks(ctx, []v1.Hook_Condition{
		v1.Hook_CONDITION_CHECK_START,
	}, HookVars{}); err != nil {
		return fmt.Errorf("check start hook: %w", err)
	}

	err = repo.UnlockIfAutoEnabled(ctx)
	if err != nil {
		return fmt.Errorf("auto unlock repo %q: %w", t.RepoID(), err)
	}

	opCheck := &v1.Operation_OperationCheck{
		OperationCheck: &v1.OperationCheck{},
	}
	op.Op = opCheck

	checkCtx, cancelCheckCtx := context.WithCancel(ctx)
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

				if opCheck.OperationCheck.Output != string(output) {
					opCheck.OperationCheck.Output = string(output)

					if err := runner.OpLog().Update(op); err != nil {
						zap.L().Error("update check operation with status output", zap.Error(err))
					}
				}
			case <-checkCtx.Done():
				return
			}
		}
	}()

	err = repo.Check(checkCtx, bufWriter)
	cancelCheckCtx()
	wg.Wait()
	if err != nil {
		runner.ExecuteHooks(ctx, []v1.Hook_Condition{
			v1.Hook_CONDITION_CHECK_ERROR,
			v1.Hook_CONDITION_ANY_ERROR,
		}, HookVars{
			Error: err.Error(),
		})

		return fmt.Errorf("check: %w", err)
	}

	opCheck.OperationCheck.Output = string(buf.Bytes())

	if err := runner.ExecuteHooks(ctx, []v1.Hook_Condition{
		v1.Hook_CONDITION_CHECK_SUCCESS,
	}, HookVars{}); err != nil {
		return fmt.Errorf("execute check success hooks: %w", err)
	}

	return nil
}
