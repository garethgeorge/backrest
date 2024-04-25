package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/config"
	"github.com/garethgeorge/backrest/internal/hook"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/queue"
	"github.com/garethgeorge/backrest/internal/rotatinglog"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

var ErrRepoNotFound = errors.New("repo not found")
var ErrRepoInitializationFailed = errors.New("repo initialization failed")
var ErrPlanNotFound = errors.New("plan not found")

const PlanForUnassociatedOperations = "_unassociated_"

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
	taskQueue    *queue.TimePriorityQueue[ScheduledTask]
	hookExecutor *hook.HookExecutor
	logStore     *rotatinglog.RotatingLog

	runningTask ScheduledTask

	// now for the purpose of testing; used by Run() to get the current time.
	now func() time.Time
}

func NewOrchestrator(resticBin string, cfg *v1.Config, oplog *oplog.OpLog, logStore *rotatinglog.RotatingLog) (*Orchestrator, error) {
	cfg = proto.Clone(cfg).(*v1.Config)

	// create the orchestrator.
	var o *Orchestrator
	o = &Orchestrator{
		OpLog:  oplog,
		config: cfg,
		// repoPool created with a memory store to ensure the config is updated in an atomic operation with the repo pool's config value.
		repoPool:     newResticRepoPool(resticBin, &config.MemoryStore{Config: cfg}),
		taskQueue:    queue.NewTimePriorityQueue[ScheduledTask](),
		hookExecutor: hook.NewHookExecutor(oplog, logStore),
		logStore:     logStore,
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

	// Update the config provided to the repo pool which is cached and diffed separately.
	if err := o.repoPool.configProvider.Update(cfg); err != nil {
		return fmt.Errorf("failed to update repo pool config: %w", err)
	}

	return o.ScheduleDefaultTasks(cfg)
}

// rescheduleTasksIfNeeded checks if any tasks need to be rescheduled based on config changes.
func (o *Orchestrator) ScheduleDefaultTasks(config *v1.Config) error {
	zap.L().Info("scheduling default tasks, waiting for task queue reset.")
	removedTasks := o.taskQueue.Reset()
	for _, t := range removedTasks {
		if err := t.cancel(o.OpLog); err != nil {
			zap.L().Error("failed to cancel queued task", zap.String("task", t.Task.Name()), zap.Error(err))
		} else {
			zap.L().Debug("queued task cancelled due to config change", zap.String("task", t.Task.Name()))
		}
	}

	zap.L().Info("reset task queue, scheduling new task set.")

	// Requeue tasks that are affected by the config change.
	o.ScheduleTask(&tasks.CollectGarbageTask{
		orchestrator: o,
	}, TaskPriorityDefault)

	for _, plan := range config.Plans {
		if plan.Disabled {
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

func (o *Orchestrator) GetRepoOrchestrator(repoId string) (repo *RepoOrchestrator, err error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	r, err := o.repoPool.GetRepo(repoId)
	if err != nil {
		return nil, fmt.Errorf("get repo %q: %w", repoId, err)
	}
	return r, nil
}

func (o *Orchestrator) GetRepo(repoId string) (*v1.Repo, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	for _, r := range o.config.Repos {
		if r.GetId() == repoId {
			return r, nil
		}
	}

	return nil, ErrRepoNotFound
}

func (o *Orchestrator) GetPlan(planId string) (*v1.Plan, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

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

	if o.runningTask != nil && o.runningTask.Op.Id == operationId {
		if err := o.runningTask.Cancel(status); err != nil {
			return fmt.Errorf("cancel running task %q: %w", o.runningTask.Name(), err)
		}
		return nil
	}

	allTasks := o.taskQueue.GetAll()
	idx := slices.IndexFunc(allTasks, func(t scheduledTask) bool {
		return t.task.OperationId() == operationId
	})
	if idx == -1 {
		return nil
	}

	t := allTasks[idx]
	o.taskQueue.Remove(t)
	if err := t.task.Cancel(status); err != nil {
		return fmt.Errorf("cancel task %q: %w", t.task.Name(), err)
	}
	if nextTime := t.task.Next(t.runAt.Add(1 * time.Second)); nextTime != nil {
		t.runAt = *nextTime
		o.taskQueue.Enqueue(*nextTime, t.priority, t)
	}
	return nil
}

// Run is the main orchestration loop. Cancel the context to stop the loop.
func (o *Orchestrator) Run(ctx context.Context) {
	zap.L().Info("starting orchestrator loop")

	for {
		if ctx.Err() != nil {
			zap.L().Info("shutting down orchestrator loop, context cancelled.")
			break
		}

		t := o.taskQueue.Dequeue(ctx)
		if t.task == nil {
			continue
		}

		zap.L().Info("running task", zap.String("task", t.task.Name()))

		o.mu.Lock()
		if o.runningTask != nil {
			panic("running task already set")
		}
		o.runningTask = t.task
		o.mu.Unlock()

		start := time.Now()
		err := t.task.Run(ctx)
		if err != nil {
			zap.L().Error("task failed", zap.String("task", t.task.Name()), zap.Error(err), zap.Duration("duration", time.Since(start)))
		} else {
			zap.L().Info("task finished", zap.String("task", t.task.Name()), zap.Duration("duration", time.Since(start)))
		}

		o.mu.Lock()
		o.runningTask = nil
		if t.config == o.config {
			// Only reschedule tasks if the config hasn't changed since the task was scheduled.
			o.ScheduleTask(t.task, t.priority)
		}
		o.mu.Unlock()

		go func() {
			for _, cb := range t.callbacks {
				cb(err)
			}
		}()
	}
}

func (o *Orchestrator) ScheduleTask(t Task, priority int, callbacks ...func(error)) {
	nextRun := t.Next(o.curTime())
	if nextRun == nil {
		return
	}
	zap.L().Info("scheduling task", zap.String("task", t.Name()), zap.String("runAt", nextRun.Format(time.RFC3339)))
	o.taskQueue.Enqueue(*nextRun, priority, scheduledTask{
		task:      t,
		runAt:     *nextRun,
		priority:  priority,
		callbacks: callbacks,
		config:    o.config,
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

	// Otherwise create a new repo.
	repo, err = NewRepoOrchestrator(repoProto, rp.resticPath)
	if err != nil {
		return nil, err
	}
	rp.repos[repoId] = repo
	return repo, nil
}

type taskExecutionInfo struct {
	operationId int64
	cancel      func()
}
