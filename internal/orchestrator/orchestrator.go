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
	"github.com/garethgeorge/backrest/internal/oplog/indexutil"
	"github.com/garethgeorge/backrest/internal/orchestrator/repo"
	"github.com/garethgeorge/backrest/internal/orchestrator/tasks"
	"github.com/garethgeorge/backrest/internal/queue"
	"github.com/garethgeorge/backrest/internal/rotatinglog"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

var ErrRepoNotFound = errors.New("repo not found")
var ErrRepoInitializationFailed = errors.New("repo initialization failed")
var ErrPlanNotFound = errors.New("plan not found")

// Orchestrator is responsible for managing repos and backups.
type Orchestrator struct {
	mu           sync.Mutex
	config       *v1.Config
	OpLog        *oplog.OpLog
	repoPool     *resticRepoPool
	taskQueue    *queue.TimePriorityQueue[stContainer]
	hookExecutor *hook.HookExecutor
	logStore     *rotatinglog.RotatingLog

	// cancelNotify is a list of channels that are notified when a task should be cancelled.
	cancelNotify []chan int64

	// now for the purpose of testing; used by Run() to get the current time.
	now func() time.Time
}

type stContainer struct {
	tasks.ScheduledTask
	configModno int32
	callbacks   []func(error)
}

func (st stContainer) Eq(other stContainer) bool {
	return st.ScheduledTask.Eq(other.ScheduledTask)
}

func (st stContainer) Less(other stContainer) bool {
	return st.ScheduledTask.Less(other.ScheduledTask)
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
		taskQueue:    queue.NewTimePriorityQueue[stContainer](),
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
			repo, err := o.GetRepoOrchestrator(repoId)
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

	zap.L().Info("orchestrator created")

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
		if t.Op == nil {
			continue
		}

		if err := o.cancelHelper(t.Op, v1.OperationStatus_STATUS_SYSTEM_CANCELLED); err != nil {
			zap.L().Error("failed to cancel queued task", zap.String("task", t.Task.Name()), zap.Error(err))
		} else {
			zap.L().Debug("queued task cancelled due to config change", zap.String("task", t.Task.Name()))
		}
	}

	zap.L().Info("reset task queue, scheduling new task set.")

	// Requeue tasks that are affected by the config change.
	if err := o.scheduleTaskHelper(tasks.NewCollectGarbageTask(), tasks.TaskPriorityDefault); err != nil {
		return fmt.Errorf("schedule collect garbage task: %w", err)
	}

	for _, plan := range config.Plans {
		if plan.Disabled {
			continue
		}
		t, err := tasks.NewScheduledBackupTask(plan)
		if err != nil {
			return fmt.Errorf("schedule backup task for plan %q: %w", plan.Id, err)
		}
		if err := o.scheduleTaskHelper(t, tasks.TaskPriorityDefault); err != nil {
			return fmt.Errorf("schedule backup task for plan %q: %w", plan.Id, err)
		}
	}

	return nil
}

func (o *Orchestrator) GetRepoOrchestrator(repoId string) (repo *repo.RepoOrchestrator, err error) {
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

	return nil, fmt.Errorf("get repo %q: %w", repoId, ErrRepoNotFound)
}

func (o *Orchestrator) GetPlan(planId string) (*v1.Plan, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	for _, p := range o.config.Plans {
		if p.Id == planId {
			return p, nil
		}
	}

	return nil, fmt.Errorf("get plan %q: %w", planId, ErrPlanNotFound)
}

func (o *Orchestrator) FlowIDForSnapshotID(snapshotID string) (int64, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	var flowID int64
	if err := o.OpLog.ForEachBySnapshotId(snapshotID, indexutil.CollectAll(), func(op *v1.Operation) error {
		flowID = op.FlowId
		return nil
	}); err != nil {
		return 0, fmt.Errorf("get flow id for snapshot %q: %w", snapshotID, err)
	}

	return flowID, nil
}

func (o *Orchestrator) CancelOperation(operationId int64, status v1.OperationStatus) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	for _, c := range o.cancelNotify {
		select {
		case c <- operationId:
		default:
		}
	}

	allTasks := o.taskQueue.GetAll()
	idx := slices.IndexFunc(allTasks, func(t stContainer) bool {
		return t.Op != nil && t.Op.GetId() == operationId
	})
	if idx == -1 {
		return nil
	}
	t := allTasks[idx]

	if err := o.cancelHelper(t.Op, status); err != nil {
		return fmt.Errorf("cancel operation: %w", err)
	}

	o.taskQueue.Remove(t)
	return nil
}

func (o *Orchestrator) cancelHelper(op *v1.Operation, status v1.OperationStatus) error {
	op.Status = status
	op.UnixTimeEndMs = time.Now().UnixMilli()
	if err := o.OpLog.Update(op); err != nil {
		return fmt.Errorf("update cancelled operation: %w", err)
	}
	return nil

}

// Run is the main orchestration loop. Cancel the context to stop the loop.
func (o *Orchestrator) Run(ctx context.Context) {
	zap.L().Info("starting orchestrator loop")

	// subscribe to cancel notifications.
	o.mu.Lock()
	cancelNotifyChan := make(chan int64, 10) // buffered to queue up cancel notifications.
	o.cancelNotify = append(o.cancelNotify, cancelNotifyChan)
	o.mu.Unlock()

	defer func() {
		o.mu.Lock()
		o.cancelNotify = slices.DeleteFunc(o.cancelNotify, func(c chan int64) bool {
			return c == cancelNotifyChan
		})
		o.mu.Unlock()
	}()

	for {
		if ctx.Err() != nil {
			zap.L().Info("shutting down orchestrator loop, context cancelled.")
			break
		}

		t := o.taskQueue.Dequeue(ctx)
		if t.Task == nil {
			continue
		}

		zap.L().Info("running task", zap.String("task", t.Task.Name()))

		taskCtx, cancel := context.WithCancel(ctx)
		go func() {
			for {
				select {
				case <-taskCtx.Done():
					return
				case opID := <-cancelNotifyChan:
					if t.Op != nil && opID == t.Op.GetId() {
						cancel()
					}
				}
			}
		}()

		start := time.Now()
		runner := newTaskRunnerImpl(o, t.Task)
		err := t.Task.Run(taskCtx, t.ScheduledTask, runner)
		cancel()

		if err != nil {
			zap.L().Error("task failed", zap.String("task", t.Task.Name()), zap.Error(err), zap.Duration("duration", time.Since(start)))
		} else {
			zap.L().Info("task finished", zap.String("task", t.Task.Name()), zap.Duration("duration", time.Since(start)))
		}

		o.mu.Lock()
		if t.configModno == o.config.Modno {
			// Only reschedule tasks if the config hasn't changed since the task was scheduled.
			if err := o.scheduleTaskHelper(t.Task, tasks.TaskPriorityDefault); err != nil {
				zap.L().Error("reschedule task", zap.String("task", t.Task.Name()), zap.Error(err))
			}
			o.mu.Unlock()
		}

		go func() {
			for _, cb := range t.callbacks {
				cb(err)
			}
		}()

	}
}

func (o *Orchestrator) ScheduleTask(t tasks.Task, priority int, callbacks ...func(error)) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	return o.scheduleTaskHelper(t, priority, callbacks...)
}

func (o *Orchestrator) scheduleTaskHelper(t tasks.Task, priority int, callbacks ...func(error)) error {
	nextRun := t.Next(o.curTime(), newTaskRunnerImpl(o, t))
	if nextRun.Eq(tasks.NeverScheduledTask) {
		return nil
	}

	stc := stContainer{
		ScheduledTask: nextRun,
		configModno:   o.config.Modno,
		callbacks:     callbacks,
	}

	if stc.Op != nil {
		stc.Op.Status = v1.OperationStatus_STATUS_PENDING
		stc.Op.UnixTimeStartMs = nextRun.RunAt.UnixMilli()

		if err := o.OpLog.Add(nextRun.Op); err != nil {
			return fmt.Errorf("add operation to oplog: %w", err)
		}
	}

	zap.L().Info("scheduling task", zap.String("task", t.Name()), zap.String("runAt", nextRun.RunAt.Format(time.RFC3339)))
	o.taskQueue.Enqueue(nextRun.RunAt, priority, stc)
	return o.ScheduleTask(t, tasks.TaskPriorityDefault, callbacks...)
}

// resticRepoPool caches restic repos.
type resticRepoPool struct {
	mu             sync.Mutex
	resticPath     string
	repos          map[string]*repo.RepoOrchestrator
	configProvider config.ConfigStore
}

func newResticRepoPool(resticPath string, configProvider config.ConfigStore) *resticRepoPool {
	return &resticRepoPool{
		resticPath:     resticPath,
		repos:          make(map[string]*repo.RepoOrchestrator),
		configProvider: configProvider,
	}
}

func (rp *resticRepoPool) GetRepo(repoId string) (*repo.RepoOrchestrator, error) {
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
	r, ok := rp.repos[repoId]
	if ok && proto.Equal(r.Config(), repoProto) {
		return r, nil
	}
	delete(rp.repos, repoId)

	// Otherwise create a new repo.
	r, err = repo.NewRepoOrchestrator(repoProto, rp.resticPath)
	if err != nil {
		return nil, err
	}
	rp.repos[repoId] = r
	return r, nil
}

type taskExecutionInfo struct {
	operationId int64
	cancel      func()
}
