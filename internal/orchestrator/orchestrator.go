package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/config"
	"github.com/garethgeorge/backrest/internal/hook"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/rotatinglog"
	"github.com/garethgeorge/backrest/pkg/restic"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

var ErrRepoNotFound = errors.New("repo not found")
var ErrRepoInitializationFailed = errors.New("repo initialization failed")
var ErrPlanNotFound = errors.New("plan not found")

const (
	TaskPriorityDefault        = 0
	TaskPriorityInteractive    = 10
	TaskPriorityIndexSnapshots = 101
	TaskPriorityForget         = 102
	TaskPriorityPrune          = 103
	TaskPriorityHook           = 1000 // runs before any other task.
	TaskPriorityStats          = -1   // very low priority.
)

// Orchestrator is responsible for managing repos and backups.
type Orchestrator struct {
	mu           sync.Mutex
	config       *v1.Config
	OpLog        *oplog.OpLog
	repoPool     *resticRepoPool
	taskQueue    taskQueue
	hookExecutor *hook.HookExecutor

	// now for the purpose of testing; used by Run() to get the current time.
	now func() time.Time

	runningTask atomic.Pointer[taskExecutionInfo]
}

func NewOrchestrator(resticBin string, cfg *v1.Config, oplog *oplog.OpLog, logStore *rotatinglog.RotatingLog) (*Orchestrator, error) {
	cfg = proto.Clone(cfg).(*v1.Config)

	// create the orchestrator.
	var o *Orchestrator
	o = &Orchestrator{
		OpLog:  oplog,
		config: cfg,
		// repoPool created with a memory store to ensure the config is updated in an atomic operation with the repo pool's config value.
		repoPool: newResticRepoPool(resticBin, &config.MemoryStore{Config: cfg}),
		taskQueue: newTaskQueue(func() time.Time {
			return o.curTime()
		}),
		hookExecutor: hook.NewHookExecutor(oplog, logStore),
	}

	// verify the operation log and mark any incomplete operations as failed.
	if oplog != nil { // oplog may be nil for testing.
		var incompleteOpRepos []string
		if err := oplog.Scan(func(incomplete *v1.Operation) {
			incomplete.Status = v1.OperationStatus_STATUS_ERROR
			incomplete.DisplayMessage = "Failed, orchestrator killed while operation was in progress."

			if incomplete.RepoId != "" && !slices.Contains(incompleteOpRepos, incomplete.RepoId) {
				incompleteOpRepos = append(incompleteOpRepos, incomplete.RepoId)
			}
		}); err != nil {
			return nil, fmt.Errorf("scan oplog: %w", err)
		}

		for _, repoId := range incompleteOpRepos {
			repo, err := o.GetRepo(repoId)
			if err != nil {
				if errors.Is(err, ErrRepoNotFound) {
					zap.L().Warn("repo not found for incomplete operation. Possibly just deleted.", zap.String("repo", repoId))
				}
				return nil, fmt.Errorf("get repo %q: %w", repoId, err)
			}

			if err := repo.Unlock(context.Background()); err != nil {
				zap.L().Error("failed to unlock repo", zap.String("repo", repoId), zap.Error(err))
			}
		}
	}

	// apply starting configuration which also queues initial tasks.
	if err := o.ApplyConfig(cfg); err != nil {
		return nil, fmt.Errorf("apply initial config: %w", err)
	}

	return o, nil
}

func (o *Orchestrator) curTime() time.Time {
	if o.now != nil {
		return o.now()
	}
	return time.Now()
}

func (o *Orchestrator) ApplyConfig(cfg *v1.Config) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.config = cfg

	// Update the config provided to the repo pool.
	if err := o.repoPool.configProvider.Update(cfg); err != nil {
		return fmt.Errorf("failed to update repo pool config: %w", err)
	}

	// reset queued tasks, this may loose any ephemeral operations scheduled by RPC. Tasks in progress aren't returned by Reset() so they will not be cancelled.
	zap.L().Info("Applying config to orchestrator, waiting for task queue reset.")
	removedTasks := o.taskQueue.Reset()
	for _, t := range removedTasks {
		if err := t.task.Cancel(v1.OperationStatus_STATUS_SYSTEM_CANCELLED); err != nil {
			zap.L().Error("failed to cancel queued task", zap.String("task", t.task.Name()), zap.Error(err))
		} else {
			zap.L().Debug("queued task cancelled due to config change", zap.String("task", t.task.Name()))
		}
	}
	zap.L().Info("Applied config to orchestrator, task queue reset. Rescheduling planned tasks now.")

	// Requeue tasks that are affected by the config change.
	o.ScheduleTask(&CollectGarbageTask{
		orchestrator: o,
	}, TaskPriorityDefault)
	for _, plan := range cfg.Plans {
		if plan.Cron == "" {
			continue
		}
		t, err := NewScheduledBackupTask(o, plan)
		if err != nil {
			return fmt.Errorf("schedule backup task for plan %q: %w", plan.Id, err)
		}
		o.ScheduleTask(t, TaskPriorityDefault)
	}

	return nil
}

func (o *Orchestrator) GetRepo(repoId string) (repo *RepoOrchestrator, err error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	r, err := o.repoPool.GetRepo(repoId)
	if err != nil {
		return nil, fmt.Errorf("get repo %q: %w", repoId, err)
	}
	return r, nil
}

func (o *Orchestrator) GetPlan(planId string) (*v1.Plan, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.config.Plans == nil {
		return nil, ErrPlanNotFound
	}

	for _, p := range o.config.Plans {
		if p.Id == planId {
			return p, nil
		}
	}

	return nil, ErrPlanNotFound
}

func (o *Orchestrator) CancelOperation(operationId int64, status v1.OperationStatus) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	// note: if the task is running the requested status will not be set.
	if running := o.runningTask.Load(); running != nil && running.operationId == operationId {
		running.cancel()
	}

	tasks := o.taskQueue.Reset()
	remaining := make([]scheduledTask, 0, len(tasks))

	for _, t := range tasks {
		if t.task.OperationId() == operationId {
			if err := t.task.Cancel(status); err != nil {
				return fmt.Errorf("cancel task %q: %w", t.task.Name(), err)
			}

			// check if the task has a next after it's current 'runAt' time, if it does then we will schedule the next run.
			if nextTime := t.task.Next(t.runAt); nextTime != nil {
				remaining = append(remaining, scheduledTask{
					task:  t.task,
					runAt: *nextTime,
				})
			}
		} else {
			remaining = append(remaining, *t)
		}
	}

	o.taskQueue.Push(remaining...)

	return nil
}

// Run is the main orchestration loop. Cancel the context to stop the loop.
func (o *Orchestrator) Run(mainCtx context.Context) {
	zap.L().Info("starting orchestrator loop")

	for {
		if mainCtx.Err() != nil {
			zap.L().Info("shutting down orchestrator loop, context cancelled.")
			break
		}

		t := o.taskQueue.Dequeue(mainCtx)
		if t == nil {
			continue
		}

		zap.L().Info("running task", zap.String("task", t.task.Name()))

		taskCtx, cancel := context.WithCancel(mainCtx)

		if swapped := o.runningTask.CompareAndSwap(nil, &taskExecutionInfo{
			operationId: t.task.OperationId(),
			cancel:      cancel,
		}); !swapped {
			zap.L().Fatal("failed to start task, another task is already running. Was Run() called twice?")
		}

		start := time.Now()
		err := t.task.Run(taskCtx)
		if err != nil {
			zap.L().Error("task failed", zap.String("task", t.task.Name()), zap.Error(err), zap.Duration("duration", time.Since(start)))
		} else {
			zap.L().Info("task finished", zap.String("task", t.task.Name()), zap.Duration("duration", time.Since(start)))
		}
		o.runningTask.Store(nil)

		for _, cb := range t.callbacks {
			cb(err)
		}

		if nextTime := t.task.Next(o.curTime()); nextTime != nil {
			o.taskQueue.Push(scheduledTask{
				task:  t.task,
				runAt: *nextTime,
			})
		}
	}
}

func (o *Orchestrator) ScheduleTask(t Task, priority int, callbacks ...func(error)) {
	nextRun := t.Next(o.curTime())
	if nextRun == nil {
		return
	}
	zap.L().Info("scheduling task", zap.String("task", t.Name()), zap.String("runAt", nextRun.Format(time.RFC3339)))
	o.taskQueue.Push(scheduledTask{
		task:      t,
		runAt:     *nextRun,
		priority:  priority,
		callbacks: callbacks,
	})
}

// resticRepoPool caches restic repos.
type resticRepoPool struct {
	mu             sync.Mutex
	resticPath     string
	repos          map[string]*RepoOrchestrator
	configProvider config.ConfigStore
}

func newResticRepoPool(resticPath string, configProvider config.ConfigStore) *resticRepoPool {
	return &resticRepoPool{
		resticPath:     resticPath,
		repos:          make(map[string]*RepoOrchestrator),
		configProvider: configProvider,
	}
}

func (rp *resticRepoPool) GetRepo(repoId string) (repo *RepoOrchestrator, err error) {
	cfg, err := rp.configProvider.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	rp.mu.Lock()
	defer rp.mu.Unlock()

	if cfg.Repos == nil {
		return nil, ErrRepoNotFound
	}

	var repoProto *v1.Repo
	for _, r := range cfg.Repos {
		if r.GetId() == repoId {
			repoProto = r
		}
	}

	if repoProto == nil {
		return nil, ErrRepoNotFound
	}

	// Check if we already have a repo for this id, if we do return it.
	repo, ok := rp.repos[repoId]
	if ok && proto.Equal(repo.repoConfig, repoProto) {
		return repo, nil
	}
	delete(rp.repos, repoId)

	var opts []restic.GenericOption
	opts = append(opts, restic.WithPropagatedEnvVars(restic.EnvToPropagate...))
	if len(repoProto.GetEnv()) > 0 {
		opts = append(opts, restic.WithEnv(repoProto.GetEnv()...))
	}
	if len(repoProto.GetFlags()) > 0 {
		opts = append(opts, restic.WithFlags(repoProto.GetFlags()...))
	}

	// Otherwise create a new repo.
	repo = newRepoOrchestrator(repoProto, restic.NewRepo(rp.resticPath, repoProto, opts...))
	rp.repos[repoId] = repo
	return repo, nil
}

type taskExecutionInfo struct {
	operationId int64
	cancel      func()
}
