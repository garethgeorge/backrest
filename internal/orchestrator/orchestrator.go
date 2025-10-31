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
	"github.com/garethgeorge/backrest/internal/logstore"
	"github.com/garethgeorge/backrest/internal/metric"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/orchestrator/logging"
	"github.com/garethgeorge/backrest/internal/orchestrator/repo"
	"github.com/garethgeorge/backrest/internal/orchestrator/tasks"
	"github.com/garethgeorge/backrest/internal/queue"
	"github.com/google/uuid"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

var ErrRepoNotFound = errors.New("repo not found")
var ErrRepoInitializationFailed = errors.New("repo initialization failed")
var ErrPlanNotFound = errors.New("plan not found")

const (
	defaultTaskLogDuration = 14 * 24 * time.Hour

	defaultAutoInitTimeout = 60 * time.Second
)

// Orchestrator is responsible for managing repos and backups.
type Orchestrator struct {
	mu                 sync.Mutex
	configMgr          *config.ConfigManager
	config             *v1.Config
	OpLog              *oplog.OpLog
	repoPool           *resticRepoPool
	taskQueue          *queue.TimePriorityQueue[stContainer]
	lastQueueResetTime time.Time
	logStore           *logstore.LogStore
	resticBin          string

	taskCancelMu sync.Mutex
	taskCancel   map[int64]context.CancelFunc

	// now for the purpose of testing; used by Run() to get the current time.
	now func() time.Time
}

var _ tasks.TaskExecutor = &Orchestrator{}

type stContainer struct {
	tasks.ScheduledTask
	retryCount  int // number of times this task has been retried.
	createdTime time.Time
	callbacks   []func(error)
}

func (st stContainer) Eq(other stContainer) bool {
	return st.ScheduledTask.Eq(other.ScheduledTask)
}

func (st stContainer) Less(other stContainer) bool {
	return st.ScheduledTask.Less(other.ScheduledTask)
}

func NewOrchestrator(resticBin string, cfgMgr *config.ConfigManager, log *oplog.OpLog, logStore *logstore.LogStore) (*Orchestrator, error) {
	// create the orchestrator.
	o := &Orchestrator{
		OpLog:      log,
		configMgr:  cfgMgr,
		taskQueue:  queue.NewTimePriorityQueue[stContainer](),
		logStore:   logStore,
		taskCancel: make(map[int64]context.CancelFunc),
		resticBin:  resticBin,
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

		if len(incompleteRepos) > 0 {
			defer func() {
				for _, repoId := range incompleteRepos {
					repo, err := o.GetRepoOrchestrator(repoId)
					if err != nil {
						if errors.Is(err, ErrRepoNotFound) {
							zap.L().Warn("repo not found for incomplete operation. Possibly just deleted.", zap.String("repo", repoId))
						}
						zap.L().Error("failed to unlock repo", zap.String("repo", repoId), zap.Error(err))
						continue
					}

					if err := repo.Unlock(context.Background()); err != nil {
						zap.L().Error("failed to unlock repo", zap.String("repo", repoId), zap.Error(err))
					}
				}
			}()
		}

		zap.L().Info("scrubbed operation log for incomplete operations",
			zap.Duration("duration", time.Since(startTime)),
			zap.Int("incomplete_ops", len(incompleteOps)),
			zap.Int("incomplete_repos", len(incompleteRepos)),
			zap.Int("deleted_ops", len(toDelete)))
	}

	// initialize any uninitialized repos.
	if err := o.autoInitReposIfNeeded(resticBin); err != nil {
		return nil, fmt.Errorf("auto-initialize repos: %w", err)
	}

	// apply starting configuration which also queues initial tasks.
	cfg, err := o.configMgr.Get()
	if err != nil {
		return nil, fmt.Errorf("get config: %w", err)
	}
	if err := o.applyConfig(cfg); err != nil {
		return nil, fmt.Errorf("apply initial config: %w", err)
	}

	zap.L().Info("orchestrator created")

	return o, nil
}

func (o *Orchestrator) autoInitReposIfNeeded(resticBin string) error {
	var fullErr error
	cfg, err := o.configMgr.Get()
	if err != nil {
		return fmt.Errorf("get config: %w", err)
	}
	cfg = proto.Clone(cfg).(*v1.Config)

	initializedRepo := false
	for _, r := range cfg.Repos {
		if !r.AutoInitialize {
			continue
		}
		zap.L().Info("auto-initializing repo", zap.String("repo", r.Id))
		rorch, err := repo.NewRepoOrchestrator(cfg, r, resticBin)
		if err != nil {
			fullErr = multierr.Append(fullErr, fmt.Errorf("auto-initialize repo %q: %w", r.Id, err))
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), defaultAutoInitTimeout)
		defer cancel()
		if err := rorch.Init(ctx); err != nil {
			fullErr = multierr.Append(fullErr, fmt.Errorf("auto-initialize repo %q: %w", r.Id, err))
			continue
		}

		guid, err := rorch.RepoGUID()
		if err != nil {
			return fmt.Errorf("get repo %q guid: %w", r.Id, err)
		}
		r.Guid = guid
		r.AutoInitialize = false
		initializedRepo = true
	}

	if initializedRepo && fullErr == nil {
		cfg.Modno++
		if err := o.configMgr.Update(cfg); err != nil {
			return fmt.Errorf("update config: %w", err)
		}
	}

	return fullErr
}

func (o *Orchestrator) curTime() time.Time {
	if o.now != nil {
		return o.now()
	}
	return time.Now()
}

func (o *Orchestrator) applyConfig(cfg *v1.Config) error {
	for _, repo := range cfg.Repos {
		if repo.Guid == "" {
			return errors.New("guid is required for all repos")
		}
	}
	o.mu.Lock()
	o.config = proto.Clone(cfg).(*v1.Config)
	o.repoPool = newResticRepoPool(o.resticBin, o.config)
	o.mu.Unlock()
	return o.ScheduleDefaultTasks(cfg)
}

// rescheduleTasksIfNeeded checks if any tasks need to be rescheduled based on config changes.
func (o *Orchestrator) ScheduleDefaultTasks(config *v1.Config) error {
	if o.OpLog == nil {
		return nil
	}

	zap.L().Info("scheduling default tasks, waiting for task queue reset.")
	o.mu.Lock()
	removedTasks := o.taskQueue.Reset()
	o.lastQueueResetTime = o.curTime()
	o.mu.Unlock()

	ids := []int64{}
	for _, t := range removedTasks {
		if t.Op.GetId() != 0 {
			ids = append(ids, t.Op.GetId())
		}
	}
	if err := o.OpLog.Delete(ids...); err != nil {
		zap.S().Warnf("failed to delete cancelled tasks from oplog: %v", err)
	}

	zap.L().Info("reset task queue, scheduling new task set", zap.String("timezone", time.Now().Location().String()))

	// Requeue tasks that are affected by the config change.
	if err := o.ScheduleTask(tasks.NewCollectGarbageTask(o.logStore), tasks.TaskPriorityDefault); err != nil {
		return fmt.Errorf("schedule collect garbage task: %w", err)
	}

	var repoByID = map[string]*v1.Repo{}
	for _, repo := range config.Repos {
		repoByID[repo.GetId()] = repo
	}

	for _, plan := range config.Plans {
		// Schedule a backup task for the plan
		repo := repoByID[plan.Repo]
		if repo == nil {
			return fmt.Errorf("repo %q not found for plan %q", plan.Repo, plan.Id)
		}

		t := tasks.NewScheduledBackupTask(repo, plan)
		if err := o.ScheduleTask(t, tasks.TaskPriorityDefault); err != nil {
			return fmt.Errorf("schedule backup task for plan %q: %w", plan.Id, err)
		}
	}

	for _, repo := range config.Repos {
		// Schedule a prune task for the repo
		t := tasks.NewPruneTask(repo, tasks.PlanForSystemTasks, false)
		if err := o.ScheduleTask(t, tasks.TaskPriorityPrune); err != nil {
			return fmt.Errorf("schedule prune task for repo %q: %w", repo.GetId(), err)
		}

		// Schedule a check task for the repo
		t = tasks.NewCheckTask(repo, tasks.PlanForSystemTasks, false)
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
	allTasks := o.taskQueue.GetAll()
	idx := slices.IndexFunc(allTasks, func(t stContainer) bool {
		return t.Op != nil && t.Op.GetId() == operationId
	})
	if idx == -1 {
		o.taskCancelMu.Lock()
		if cancel, ok := o.taskCancel[operationId]; ok {
			cancel()
		}
		o.taskCancelMu.Unlock()
		return nil
	}
	t := allTasks[idx]

	if err := o.cancelHelper(t.Op, status); err != nil {
		return fmt.Errorf("cancel operation: %w", err)
	}
	o.taskQueue.Remove(t)

	if st, err := o.CreateUnscheduledTask(t.Task, tasks.TaskPriorityDefault, t.RunAt); err != nil {
		return fmt.Errorf("reschedule cancelled task: %w", err)
	} else if !st.Eq(tasks.NeverScheduledTask) {
		o.taskQueue.Enqueue(st.RunAt, tasks.TaskPriorityDefault, stContainer{
			ScheduledTask: st,
			createdTime:   o.curTime(),
		})
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

	// Setup config watching
	configCh := o.configMgr.OnChange.Subscribe()
	defer o.configMgr.OnChange.Unsubscribe(configCh)

	// Start the config watcher goroutine
	go o.watchConfigChanges(ctx, configCh)

	// Start the clock jump detector goroutine
	go o.watchForClockJumps(ctx)

	// Main task processing loop
	for {
		if ctx.Err() != nil {
			zap.L().Info("shutting down orchestrator loop, context cancelled.")
			break
		}

		t := o.taskQueue.Dequeue(ctx)
		if t.Task == nil {
			continue
		}

		// Clone the operation in case we need to reset changes and reschedule the task for a retry
		originalOp := proto.Clone(t.Op).(*v1.Operation)
		o.prepareOperationForRetry(&t)

		// Execute the task
		err := o.RunTask(ctx, t.ScheduledTask)

		// Handle task completion, including potential retry
		if o.handleTaskCompletion(&t, err, originalOp) {
			continue // Skip callbacks for retried tasks
		}

		// Execute callbacks
		for _, cb := range t.callbacks {
			go cb(err)
		}
	}
}

// watchConfigChanges handles configuration updates from the config manager
func (o *Orchestrator) watchConfigChanges(ctx context.Context, configCh <-chan struct{}) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-configCh:
			cfg, err := o.configMgr.Get()
			if err != nil {
				zap.S().Errorf("orchestrator failed to refresh config after change notification: %v", err)
				continue
			}
			if err := o.applyConfig(cfg); err != nil {
				zap.S().Errorf("orchestrator failed to apply config: %v", err)
			}
		}
	}
}

// watchForClockJumps detects system clock jumps and reschedules tasks when detected
func (o *Orchestrator) watchForClockJumps(ctx context.Context) {
	// Watchdog timer to detect clock jumps and reschedule all tasks
	interval := 5 * time.Minute
	grace := 30 * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	lastTickTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			deltaMs := lastTickTime.Add(interval).UnixMilli() - time.Now().UnixMilli()
			lastTickTime = time.Now()
			if deltaMs < 0 {
				deltaMs = -deltaMs
			}
			if deltaMs < grace.Milliseconds() {
				continue
			}
			zap.S().Warnf("detected a clock jump, watchdog timer is off from realtime by %dms, rescheduling all tasks", deltaMs)

			if err := o.ScheduleDefaultTasks(o.config); err != nil {
				zap.S().Errorf("failed to schedule default tasks: %v", err)
			}
		}
	}
}

// prepareOperationForRetry prepares a task's operation for a retry by updating its display message
// and cleaning up previous hook executions if necessary
func (o *Orchestrator) prepareOperationForRetry(t *stContainer) {
	if t.Op == nil || t.retryCount == 0 {
		return
	}

	t.Op.DisplayMessage = fmt.Sprintf("running after %d retries", t.retryCount)

	// Delete any previous hook executions for this operation in case this is a retry
	prevHookExecutionIDs := []int64{}
	if err := o.OpLog.Query(oplog.Query{FlowID: &t.Op.FlowId}, func(op *v1.Operation) error {
		if hookOp, ok := op.Op.(*v1.Operation_OperationRunHook); ok && hookOp.OperationRunHook.GetParentOp() == t.Op.Id {
			prevHookExecutionIDs = append(prevHookExecutionIDs, op.Id)
		}
		return nil
	}); err != nil {
		zap.L().Error("failed to collect previous hook execution IDs", zap.Error(err))
	}

	if len(prevHookExecutionIDs) > 0 {
		zap.S().Debugf("deleting previous hook execution IDs: %v", prevHookExecutionIDs)
		if err := o.OpLog.Delete(prevHookExecutionIDs...); err != nil {
			zap.L().Error("failed to delete previous hook execution IDs", zap.Error(err))
		}
	}
}

// handleTaskCompletion processes task completion, handling retry logic or rescheduling as needed.
// Returns true if the task was requeued for retry.
func (o *Orchestrator) handleTaskCompletion(t *stContainer, err error, originalOp *v1.Operation) bool {
	// Check if config has changed since the task was scheduled
	o.mu.Lock()
	lastQueueResetTime := o.lastQueueResetTime
	if t.createdTime.Before(lastQueueResetTime) {
		o.mu.Unlock()
		zap.L().Debug("task was created before last queue reset, skipping reschedule", zap.String("task", t.Task.Name()))
		return false
	}
	o.mu.Unlock()

	// Handle task retry logic if needed
	var retryErr *tasks.TaskRetryError
	if errors.As(err, &retryErr) {
		o.retryTask(t, retryErr, originalOp)
		return true // Skip callbacks for retried tasks
	}

	// Regular rescheduling
	if e := o.ScheduleTask(t.Task, tasks.TaskPriorityDefault); e != nil {
		zap.L().Error("reschedule task", zap.String("task", t.Task.Name()), zap.Error(e))
	}

	return false
}

// retryTask sets up a task for retry with backoff
func (o *Orchestrator) retryTask(t *stContainer, retryErr *tasks.TaskRetryError, originalOp *v1.Operation) {
	t.retryCount++
	delay := retryErr.Backoff(t.retryCount)

	// Update operation state if present
	if t.Op != nil {
		t.Op = originalOp
		t.Op.DisplayMessage = fmt.Sprintf("waiting for retry, current backoff delay: %v", delay)
		t.Op.UnixTimeStartMs = t.RunAt.UnixMilli()
		if err := o.OpLog.Update(t.Op); err != nil {
			zap.S().Errorf("failed to update operation in oplog: %v", err)
		}
	}

	// Enqueue the task for retry
	t.RunAt = time.Now().Add(delay)
	o.taskQueue.Enqueue(t.RunAt, tasks.TaskPriorityDefault, *t)

	zap.L().Info("retrying task",
		zap.String("task", t.Task.Name()),
		zap.String("runAt", t.RunAt.Format(time.RFC3339)),
		zap.Duration("delay", delay))
}

func (o *Orchestrator) RunTask(parentCtx context.Context, st tasks.ScheduledTask) error {
	zap.L().Info("running task", zap.String("task", st.Task.Name()), zap.String("runAt", st.RunAt.Format(time.RFC3339)))

	// Set up context, logging, and cancellation
	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()
	ctx, logWriter := o.setupTaskContext(ctx, st.Op, cancel)
	defer o.cleanupTaskContext(ctx, st.Op, logWriter)

	// Run the task and record metrics
	err := o.executeTask(ctx, st)

	// Update operation status based on execution result
	if st.Op != nil {
		o.updateOperationStatus(ctx, st.Op, err)
	}

	return err
}

// setupTaskContext prepares the context for task execution with appropriate logging and cancellation
func (o *Orchestrator) setupTaskContext(ctx context.Context, op *v1.Operation, cancel context.CancelFunc) (context.Context, io.WriteCloser) {
	var logWriter io.WriteCloser

	if op != nil {
		// Register cancellation function
		o.taskCancelMu.Lock()
		o.taskCancel[op.Id] = cancel
		o.taskCancelMu.Unlock()

		// Set up logging
		logID := "t-" + uuid.New().String()
		var err error
		logWriter, err = o.logStore.Create(logID, op.Id, defaultTaskLogDuration)
		if err != nil {
			zap.S().Errorf("failed to create live log writer: %v", err)
		}
		ctx = logging.ContextWithWriter(ctx, logWriter)

		// Update operation status and log reference
		op.Logref = logID
		op.UnixTimeStartMs = time.Now().UnixMilli()
		if op.Status == v1.OperationStatus_STATUS_PENDING || op.Status == v1.OperationStatus_STATUS_UNKNOWN {
			op.Status = v1.OperationStatus_STATUS_INPROGRESS
		}

		// Record the operation in the oplog
		o.recordOperationInOplog(op)
	} else {
		// If no operation is provided, discard logs
		ctx = logging.ContextWithWriter(ctx, io.Discard)
	}

	return ctx, logWriter
}

// recordOperationInOplog adds or updates an operation in the operation log
func (o *Orchestrator) recordOperationInOplog(op *v1.Operation) {
	var err error
	if op.Id != 0 {
		err = o.OpLog.Update(op)
	} else {
		err = o.OpLog.Add(op)
	}

	if err != nil {
		zap.S().Errorf("failed to update operation in oplog: %v", err)
	}
}

// cleanupTaskContext handles cleanup after task execution
func (o *Orchestrator) cleanupTaskContext(ctx context.Context, op *v1.Operation, logWriter io.WriteCloser) {
	// Remove the cancel function from the map if there was an operation
	if op != nil {
		o.taskCancelMu.Lock()
		delete(o.taskCancel, op.Id)
		o.taskCancelMu.Unlock()
	}

	// Close the log writer if one was created
	if logWriter != nil {
		if err := logWriter.Close(); err != nil {
			zap.S().Warnf("failed to close log writer, logs may be partial: %v", err)
		}
	}
}

// executeTask runs the task and records metrics for it
func (o *Orchestrator) executeTask(ctx context.Context, st tasks.ScheduledTask) error {
	start := time.Now()
	runner := newTaskRunnerImpl(o, st.Task, st.Op)
	err := st.Task.Run(ctx, st, runner)

	// Record metrics based on task result
	if err != nil {
		runner.Logger(ctx).Error("task failed", zap.Error(err), zap.Duration("duration", time.Since(start)))
		metric.GetRegistry().RecordTaskRun(st.Task.RepoID(), st.Task.PlanID(), st.Task.Type(), time.Since(start).Seconds(), "failed")
	} else {
		runner.Logger(ctx).Info("task finished", zap.Duration("duration", time.Since(start)))
		metric.GetRegistry().RecordTaskRun(st.Task.RepoID(), st.Task.PlanID(), st.Task.Type(), time.Since(start).Seconds(), "success")
	}

	return err
}

// updateOperationStatus updates the operation's status based on the task execution result
func (o *Orchestrator) updateOperationStatus(ctx context.Context, op *v1.Operation, err error) {
	if err != nil {
		// Handle different error types
		var taskCancelledError *tasks.TaskCancelledError
		var taskRetryError *tasks.TaskRetryError
		if errors.As(err, &taskCancelledError) {
			op.Status = v1.OperationStatus_STATUS_USER_CANCELLED
		} else if errors.As(err, &taskRetryError) {
			op.Status = v1.OperationStatus_STATUS_PENDING
		} else {
			op.Status = v1.OperationStatus_STATUS_ERROR
		}

		// Prepend the error to the display message
		if op.DisplayMessage != "" {
			op.DisplayMessage = err.Error() + "\n\n" + op.DisplayMessage
		} else {
			op.DisplayMessage = err.Error()
		}

		if ctx.Err() != nil {
			op.DisplayMessage += "\n\nnote: task was interrupted by context cancellation or instance shutdown"
		}
	}

	// Set end time and update final status
	op.UnixTimeEndMs = time.Now().UnixMilli()
	if op.Status == v1.OperationStatus_STATUS_INPROGRESS {
		op.Status = v1.OperationStatus_STATUS_SUCCESS
	}

	// Update the operation in the log
	if err := o.OpLog.Update(op); err != nil {
		zap.S().Errorf("failed to update operation in oplog: %v", err)
	}
}

func (o *Orchestrator) ScheduleTask(t tasks.Task, priority int, callbacks ...func(error)) error {
	nextRun, err := o.CreateUnscheduledTask(t, priority, o.curTime())
	if err != nil {
		return err
	}
	if nextRun.Eq(tasks.NeverScheduledTask) {
		return nil
	}

	stc := stContainer{
		ScheduledTask: nextRun,
		callbacks:     callbacks,
		createdTime:   o.curTime(),
	}

	o.taskQueue.Enqueue(nextRun.RunAt, priority, stc)
	zap.L().Info("scheduled task", zap.String("task", t.Name()), zap.String("runAt", nextRun.RunAt.Format(time.RFC3339)))
	return nil
}

func (o *Orchestrator) CreateUnscheduledTask(t tasks.Task, priority int, curTime time.Time) (tasks.ScheduledTask, error) {
	nextRun, err := t.Next(curTime, newTaskRunnerImpl(o, t, nil))
	if err != nil {
		return tasks.NeverScheduledTask, fmt.Errorf("finding run time for task %q: %w", t.Name(), err)
	}
	if nextRun.Eq(tasks.NeverScheduledTask) {
		return tasks.NeverScheduledTask, nil
	}
	nextRun.Task = t

	if nextRun.Op != nil {
		nextRun.Op.InstanceId = o.config.Instance
		nextRun.Op.PlanId = t.PlanID()
		nextRun.Op.RepoId = t.Repo().GetId()
		nextRun.Op.RepoGuid = t.Repo().GetGuid()
		nextRun.Op.Status = v1.OperationStatus_STATUS_PENDING
		nextRun.Op.UnixTimeStartMs = nextRun.RunAt.UnixMilli()

		if err := o.OpLog.Add(nextRun.Op); err != nil {
			return tasks.NeverScheduledTask, fmt.Errorf("add operation to oplog: %w", err)
		}
	}
	return nextRun, nil
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
