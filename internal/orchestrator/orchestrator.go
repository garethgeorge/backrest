package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"sync"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/config"
	"github.com/garethgeorge/backrest/internal/logwriter"
	"github.com/garethgeorge/backrest/internal/metric"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/orchestrator/logging"
	"github.com/garethgeorge/backrest/internal/orchestrator/repo"
	"github.com/garethgeorge/backrest/internal/orchestrator/tasks"
	"github.com/garethgeorge/backrest/internal/queue"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

var ErrRepoNotFound = errors.New("repo not found")
var ErrRepoInitializationFailed = errors.New("repo initialization failed")
var ErrPlanNotFound = errors.New("plan not found")

// Orchestrator is responsible for managing repos and backups.
type Orchestrator struct {
	mu        sync.Mutex
	config    *v1.Config
	OpLog     *oplog.OpLog
	repoPool  *resticRepoPool
	taskQueue *queue.TimePriorityQueue[stContainer]
	logStore  *logwriter.LogManager

	// cancelNotify is a list of channels that are notified when a task should be cancelled.
	cancelNotify []chan int64

	// now for the purpose of testing; used by Run() to get the current time.
	now func() time.Time
}

var _ tasks.TaskExecutor = &Orchestrator{}

type stContainer struct {
	tasks.ScheduledTask
	retryCount  int // number of times this task has been retried.
	configModno int32
	callbacks   []func(error)
}

func (st stContainer) Eq(other stContainer) bool {
	return st.ScheduledTask.Eq(other.ScheduledTask)
}

func (st stContainer) Less(other stContainer) bool {
	return st.ScheduledTask.Less(other.ScheduledTask)
}

func NewOrchestrator(resticBin string, cfg *v1.Config, log *oplog.OpLog, logStore *logwriter.LogManager) (*Orchestrator, error) {
	cfg = proto.Clone(cfg).(*v1.Config)

	// create the orchestrator.
	o := &Orchestrator{
		OpLog:  log,
		config: cfg,
		// repoPool created with a memory store to ensure the config is updated in an atomic operation with the repo pool's config value.
		repoPool:  newResticRepoPool(resticBin, cfg),
		taskQueue: queue.NewTimePriorityQueue[stContainer](),
		logStore:  logStore,
	}

	// verify the operation log and mark any incomplete operations as failed.
	if log != nil { // oplog may be nil for testing.
		incompleteRepos := []string{}
		incompleteOps := []*v1.Operation{}
		toDelete := []int64{}

		startTime := time.Now()
		zap.S().Info("scrubbing operation log for incomplete operations")

		if err := log.Query(oplog.SelectAll, func(op *v1.Operation) error {
			if op.Status == v1.OperationStatus_STATUS_PENDING || op.Status == v1.OperationStatus_STATUS_SYSTEM_CANCELLED || op.Status == v1.OperationStatus_STATUS_USER_CANCELLED || op.Status == v1.OperationStatus_STATUS_UNKNOWN {
				toDelete = append(toDelete, op.Id)
			} else if op.Status == v1.OperationStatus_STATUS_INPROGRESS {
				incompleteOps = append(incompleteOps, op)
				if !slices.Contains(incompleteRepos, op.RepoId) {
					incompleteRepos = append(incompleteRepos, op.RepoId)
				}
			}
			return nil
		}); err != nil {
			return nil, fmt.Errorf("scan oplog: %w", err)
		}

		for _, op := range incompleteOps {
			// check for logs to finalize
			if op.Logref != "" {
				if frozenID, err := logStore.Finalize(op.Logref); err != nil {
					zap.L().Warn("failed to finalize livelog ref for incomplete operation", zap.String("logref", op.Logref), zap.Error(err))
				} else {
					op.Logref = frozenID
				}
			}
			op.Status = v1.OperationStatus_STATUS_ERROR
			op.DisplayMessage = "Operation was incomplete when orchestrator was restarted."
			op.UnixTimeEndMs = op.UnixTimeStartMs
			if err := log.Update(op); err != nil {
				return nil, fmt.Errorf("update incomplete operation: %w", err)
			}
		}

		if err := log.Delete(toDelete...); err != nil {
			return nil, fmt.Errorf("delete incomplete operations: %w", err)
		}

		for _, repoId := range incompleteRepos {
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

		for _, id := range logStore.LiveLogIDs() {
			if _, err := logStore.Finalize(id); err != nil {
				zap.L().Warn("failed to finalize unassociated live log", zap.String("id", id), zap.Error(err))
			}
		}

		zap.L().Info("scrubbed operation log for incomplete operations",
			zap.Duration("duration", time.Since(startTime)),
			zap.Int("incomplete_ops", len(incompleteOps)),
			zap.Int("incomplete_repos", len(incompleteRepos)),
			zap.Int("deleted_ops", len(toDelete)))
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
	o.config = proto.Clone(cfg).(*v1.Config)
	o.repoPool = newResticRepoPool(o.repoPool.resticPath, o.config)
	o.mu.Unlock()
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

	zap.L().Info("reset task queue, scheduling new task set", zap.String("timezone", time.Now().Location().String()))

	// Requeue tasks that are affected by the config change.
	if err := o.ScheduleTask(tasks.NewCollectGarbageTask(), tasks.TaskPriorityDefault); err != nil {
		return fmt.Errorf("schedule collect garbage task: %w", err)
	}

	for _, plan := range config.Plans {
		// Schedule a backup task for the plan
		t := tasks.NewScheduledBackupTask(plan)
		if err := o.ScheduleTask(t, tasks.TaskPriorityDefault); err != nil {
			return fmt.Errorf("schedule backup task for plan %q: %w", plan.Id, err)
		}
	}

	for _, repo := range config.Repos {
		// Schedule a prune task for the repo
		t := tasks.NewPruneTask(repo.GetId(), tasks.PlanForSystemTasks, false)
		if err := o.ScheduleTask(t, tasks.TaskPriorityPrune); err != nil {
			return fmt.Errorf("schedule prune task for repo %q: %w", repo.GetId(), err)
		}

		// Schedule a check task for the repo
		t = tasks.NewCheckTask(repo.GetId(), tasks.PlanForSystemTasks, false)
		if err := o.ScheduleTask(t, tasks.TaskPriorityCheck); err != nil {
			return fmt.Errorf("schedule check task for repo %q: %w", repo.GetId(), err)
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

func (o *Orchestrator) GetRepo(repoID string) (*v1.Repo, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	repo := config.FindRepo(o.config, repoID)
	if repo == nil {
		return nil, fmt.Errorf("get repo %q: %w", repoID, ErrRepoNotFound)
	}
	return repo, nil
}

func (o *Orchestrator) GetPlan(planID string) (*v1.Plan, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	plan := config.FindPlan(o.config, planID)
	if plan == nil {
		return nil, fmt.Errorf("get plan %q: %w", planID, ErrPlanNotFound)
	}
	return plan, nil
}

func (o *Orchestrator) CancelOperation(operationId int64, status v1.OperationStatus) error {
	o.mu.Lock()
	for _, c := range o.cancelNotify {
		select {
		case c <- operationId:
		default:
		}
	}
	o.mu.Unlock()

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

	if err := o.scheduleTaskHelper(t.Task, tasks.TaskPriorityDefault, t.RunAt); err != nil {
		return fmt.Errorf("reschedule cancelled task: %w", err)
	}
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
		if idx := slices.Index(o.cancelNotify, cancelNotifyChan); idx != -1 {
			o.cancelNotify = slices.Delete(o.cancelNotify, idx, idx+1)
		}
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

		taskCtx, cancelTaskCtx := context.WithCancel(ctx)
		go func() {
			for {
				select {
				case <-taskCtx.Done():
					return
				case opID := <-cancelNotifyChan:
					if t.Op != nil && opID == t.Op.GetId() {
						cancelTaskCtx()
					}
				}
			}
		}()

		// Clone the operation incase we need to reset changes and reschedule the task for a retry
		originalOp := proto.Clone(t.Op).(*v1.Operation)
		if t.Op != nil && t.retryCount != 0 {
			t.Op.DisplayMessage = fmt.Sprintf("running after %d retries", t.retryCount)
			// Delete any previous hook executions for this operation incase this is a retry.
			prevHookExecutionIDs := []int64{}
			if err := o.OpLog.Query(oplog.Query{FlowID: t.Op.FlowId}, func(op *v1.Operation) error {
				if hookOp, ok := op.Op.(*v1.Operation_OperationRunHook); ok && hookOp.OperationRunHook.GetParentOp() == t.Op.Id {
					prevHookExecutionIDs = append(prevHookExecutionIDs, op.Id)
				}
				return nil
			}); err != nil {
				zap.L().Error("failed to collect previous hook execution IDs", zap.Error(err))
			}
			zap.S().Debugf("deleting previous hook execution IDs: %v", prevHookExecutionIDs)
			if err := o.OpLog.Delete(prevHookExecutionIDs...); err != nil {
				zap.L().Error("failed to delete previous hook execution IDs", zap.Error(err))
			}
		}

		err := o.RunTask(taskCtx, t.ScheduledTask)

		o.mu.Lock()
		curCfgModno := o.config.Modno
		o.mu.Unlock()
		if t.configModno == curCfgModno {
			// Only reschedule tasks if the config hasn't changed since the task was scheduled.
			var retryErr *tasks.TaskRetryError
			if errors.As(err, &retryErr) {
				// If the task returned a retry error, schedule for a retry reusing the same task and operation data.
				t.retryCount += 1
				delay := retryErr.Backoff(t.retryCount)
				if t.Op != nil {
					t.Op = originalOp
					t.Op.DisplayMessage = fmt.Sprintf("waiting for retry, current backoff delay: %v", delay)
					t.Op.UnixTimeStartMs = t.RunAt.UnixMilli()
					if err := o.OpLog.Update(t.Op); err != nil {
						zap.S().Errorf("failed to update operation in oplog: %v", err)
					}
				}
				t.RunAt = time.Now().Add(delay)
				o.taskQueue.Enqueue(t.RunAt, tasks.TaskPriorityDefault, t)
				zap.L().Info("retrying task",
					zap.String("task", t.Task.Name()),
					zap.String("runAt", t.RunAt.Format(time.RFC3339)),
					zap.Duration("delay", delay))
				continue // skip executing the task's callbacks.
			} else if e := o.ScheduleTask(t.Task, tasks.TaskPriorityDefault); e != nil {
				// Schedule the next execution of the task
				zap.L().Error("reschedule task", zap.String("task", t.Task.Name()), zap.Error(e))
			}
		}
		cancelTaskCtx()

		for _, cb := range t.callbacks {
			go cb(err)
		}
	}
}

func (o *Orchestrator) RunTask(ctx context.Context, st tasks.ScheduledTask) error {
	zap.L().Info("running task", zap.String("task", st.Task.Name()), zap.String("runAt", st.RunAt.Format(time.RFC3339)))
	var liveLogID string
	var logWriter io.WriteCloser
	op := st.Op
	if op != nil {
		var err error
		liveLogID, logWriter, err = o.logStore.NewLiveWriter(fmt.Sprintf("%x", op.GetId()))
		if err != nil {
			zap.S().Errorf("failed to create live log writer: %v", err)
		}
		ctx = logging.ContextWithWriter(ctx, logWriter)

		op.Logref = liveLogID // set the logref to the live log.
		op.UnixTimeStartMs = time.Now().UnixMilli()
		if op.Status == v1.OperationStatus_STATUS_PENDING || op.Status == v1.OperationStatus_STATUS_UNKNOWN {
			op.Status = v1.OperationStatus_STATUS_INPROGRESS
		}
		if op.Id != 0 {
			if err := o.OpLog.Update(op); err != nil {
				zap.S().Errorf("failed to add operation to oplog: %w", err)
			}
		} else {
			if err := o.OpLog.Add(op); err != nil {
				zap.S().Errorf("failed to add operation to oplog: %w", err)
			}
		}
	}

	start := time.Now()
	runner := newTaskRunnerImpl(o, st.Task, st.Op)
	err := st.Task.Run(ctx, st, runner)
	if err != nil {
		runner.Logger(ctx).Error("task failed", zap.Error(err), zap.Duration("duration", time.Since(start)))
	} else {
		runner.Logger(ctx).Info("task finished", zap.Duration("duration", time.Since(start)))
		metric.GetRegistry().RecordTaskRun(st.Task.RepoID(), st.Task.PlanID(), st.Task.Type(), time.Since(start).Seconds(), "success")
	}

	if op != nil {
		// write logs to log storage for this task.
		if logWriter != nil {
			if err := logWriter.Close(); err != nil {
				zap.S().Errorf("failed to close live log writer: %v", err)
			}
			if finalID, err := o.logStore.Finalize(liveLogID); err != nil {
				zap.S().Errorf("failed to finalize live log: %v", err)
			} else {
				op.Logref = finalID
			}
		}

		if err != nil {
			var taskCancelledError *tasks.TaskCancelledError
			var taskRetryError *tasks.TaskRetryError
			if errors.As(err, &taskCancelledError) {
				op.Status = v1.OperationStatus_STATUS_USER_CANCELLED
			} else if errors.As(err, &taskRetryError) {
				op.Status = v1.OperationStatus_STATUS_PENDING
			} else {
				op.Status = v1.OperationStatus_STATUS_ERROR
			}

			// prepend the error to the display
			if op.DisplayMessage != "" {
				op.DisplayMessage = err.Error() + "\n\n" + op.DisplayMessage
			} else {
				op.DisplayMessage = err.Error()
			}
		}
		op.UnixTimeEndMs = time.Now().UnixMilli()
		if op.Status == v1.OperationStatus_STATUS_INPROGRESS {
			op.Status = v1.OperationStatus_STATUS_SUCCESS
		}
		if e := o.OpLog.Update(op); e != nil {
			zap.S().Errorf("failed to update operation in oplog: %v", e)
		}
	}

	return err
}

// ScheduleTask schedules a task to run at the next available time.
// note that o.mu must not be held when calling this function.
func (o *Orchestrator) ScheduleTask(t tasks.Task, priority int, callbacks ...func(error)) error {
	return o.scheduleTaskHelper(t, priority, o.curTime(), callbacks...)
}

func (o *Orchestrator) scheduleTaskHelper(t tasks.Task, priority int, curTime time.Time, callbacks ...func(error)) error {
	nextRun, err := t.Next(curTime, newTaskRunnerImpl(o, t, nil))
	if err != nil {
		return fmt.Errorf("finding run time for task %q: %w", t.Name(), err)
	}
	if nextRun.Eq(tasks.NeverScheduledTask) {
		return nil
	}
	nextRun.Task = t
	stc := stContainer{
		ScheduledTask: nextRun,
		configModno:   o.config.Modno,
		callbacks:     callbacks,
	}

	if stc.Op != nil {
		stc.Op.InstanceId = o.config.Instance
		stc.Op.PlanId = t.PlanID()
		stc.Op.RepoId = t.RepoID()
		stc.Op.Status = v1.OperationStatus_STATUS_PENDING
		stc.Op.UnixTimeStartMs = nextRun.RunAt.UnixMilli()

		if err := o.OpLog.Add(nextRun.Op); err != nil {
			return fmt.Errorf("add operation to oplog: %w", err)
		}
	}

	o.taskQueue.Enqueue(nextRun.RunAt, priority, stc)
	zap.L().Info("scheduled task", zap.String("task", t.Name()), zap.String("runAt", nextRun.RunAt.Format(time.RFC3339)))
	return nil
}

func (o *Orchestrator) Config() *v1.Config {
	o.mu.Lock()
	defer o.mu.Unlock()
	return proto.Clone(o.config).(*v1.Config)
}

// resticRepoPool caches restic repos.
type resticRepoPool struct {
	mu         sync.Mutex
	resticPath string
	repos      map[string]*repo.RepoOrchestrator
	config     *v1.Config
}

func newResticRepoPool(resticPath string, config *v1.Config) *resticRepoPool {
	return &resticRepoPool{
		resticPath: resticPath,
		repos:      make(map[string]*repo.RepoOrchestrator),
		config:     config,
	}
}

func (rp *resticRepoPool) GetRepo(repoId string) (*repo.RepoOrchestrator, error) {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if rp.config.Repos == nil {
		return nil, ErrRepoNotFound
	}

	// Check if we already have a repo for this id, if we do return it.
	r, ok := rp.repos[repoId]
	if ok {
		return r, nil
	}

	repoProto := config.FindRepo(rp.config, repoId)
	if repoProto == nil {
		return nil, ErrRepoNotFound
	}

	// Otherwise create a new repo.
	r, err := repo.NewRepoOrchestrator(rp.config, repoProto, rp.resticPath)
	if err != nil {
		return nil, err
	}
	rp.repos[repoId] = r
	return r, nil
}
