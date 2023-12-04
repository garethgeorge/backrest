package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	v1 "github.com/garethgeorge/resticui/gen/go/v1"
	"github.com/garethgeorge/resticui/internal/config"
	"github.com/garethgeorge/resticui/internal/oplog"
	"github.com/garethgeorge/resticui/pkg/restic"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

var ErrRepoNotFound = errors.New("repo not found")
var ErrRepoInitializationFailed = errors.New("repo initialization failed")
var ErrPlanNotFound = errors.New("plan not found")

const (
	TaskPriorityDefault = iota
	TaskPriorityIndexSnapshots
	TaskPriorityPrune
	TaskPriorityForget
	TaskPriorityInteractive // highest priority (add other priorities to this value for offsets)
)

// Orchestrator is responsible for managing repos and backups.
type Orchestrator struct {
	mu        sync.Mutex
	config    *v1.Config
	OpLog     *oplog.OpLog
	repoPool  *resticRepoPool
	taskQueue taskQueue

	// now for the purpose of testing; used by Run() to get the current time.
	now func() time.Time
}

func NewOrchestrator(resticBin string, cfg *v1.Config, oplog *oplog.OpLog) (*Orchestrator, error) {
	var o *Orchestrator
	o = &Orchestrator{
		OpLog: oplog,
		// repoPool created with a memory store to ensure the config is updated in an atomic operation with the repo pool's config value.
		repoPool: newResticRepoPool(resticBin, &config.MemoryStore{Config: cfg}),
		taskQueue: taskQueue{
			Now: func() time.Time {
				return o.curTime()
			},
		},
	}
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
	for _, plan := range cfg.Plans {
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
		if err := t.task.Run(mainCtx); err != nil {
			zap.L().Error("task failed", zap.String("task", t.task.Name()), zap.Error(err))
		} else {
			zap.L().Info("task finished", zap.String("task", t.task.Name()))
		}

		if nextTime := t.task.Next(o.curTime()); nextTime != nil {
			o.taskQueue.Push(scheduledTask{
				task:  t.task,
				runAt: *nextTime,
			})
		}
	}
}

func (o *Orchestrator) ScheduleTask(t Task, priority int) {
	nextRun := t.Next(o.curTime())
	if nextRun == nil {
		return
	}
	zap.L().Info("scheduling task", zap.String("task", t.Name()), zap.String("runAt", nextRun.Format(time.RFC3339)))
	o.taskQueue.Push(scheduledTask{
		task:     t,
		runAt:    *nextRun,
		priority: priority,
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
