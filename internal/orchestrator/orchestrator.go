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

// Orchestrator is responsible for managing repos and backups.
type Orchestrator struct {
	mu sync.Mutex
	config *v1.Config
	oplog *oplog.OpLog
	repoPool *resticRepoPool

	
	configUpdates chan *v1.Config // configUpdates chan makes config changes available to Run()
	externTasks chan Task // externTasks is a channel that externally added tasks can be added to, they will be consumed by Run()
}

func NewOrchestrator(configProvider config.ConfigStore, oplog *oplog.OpLog) (*Orchestrator, error) {
	cfg, err := configProvider.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	return &Orchestrator{
		config: cfg,
		oplog: oplog,
		repoPool: newResticRepoPool(&config.MemoryStore{Config: cfg}),
		externTasks: make(chan Task, 2),
	}, nil
}

func (o *Orchestrator) ApplyConfig(cfg *v1.Config) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.config = cfg

	zap.L().Debug("Applying config to orchestrator", zap.Any("config", cfg))
	
	// Update the config provided to the repo pool.
	if err := o.repoPool.configProvider.Update(cfg); err != nil {
		return fmt.Errorf("failed to update repo pool config: %w", err)
	}

	if o.configUpdates != nil {
		// orchestrator loop is running, notify it of the config change.
		o.configUpdates <- cfg
	}

	return nil
}

func (o *Orchestrator) GetRepo(repoId string) (repo *RepoOrchestrator, err error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	r, err := o.repoPool.GetRepo(repoId)
	if  err != nil {
		return nil, fmt.Errorf("failed to get repo %q: %w", repoId, err)
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
func (o *Orchestrator) Run(mainCtx context.Context) error {
	zap.L().Info("Starting orchestrator loop")

	o.mu.Lock()
	o.configUpdates = make(chan *v1.Config)
	o.mu.Unlock()

	for mainCtx.Err() == nil {
		o.mu.Lock()
		config := o.config
		o.mu.Unlock()
		o.runVersion(mainCtx, config)
		zap.L().Info("Restarting orchestrator loop")
	}
	zap.L().Info("Exited orchestrator loop, context cancelled.")

	return nil
}

// runImmutable is a helper function for Run() that runs the orchestration loop with a single version of the config.
func (o *Orchestrator) runVersion(mainCtx context.Context, config *v1.Config) {
	lock := sync.Mutex{}
	ctx, cancel := context.WithCancel(mainCtx)
	
	var wg sync.WaitGroup

	var execTask func(t Task)
	execTask = func(t Task) {
		curTime := time.Now()

		runAt := t.Next(curTime)
		if runAt == nil {
			zap.L().Debug("Task has no next run, not scheduling.", zap.String("task", t.Name()))
			return
		}

		timer := time.NewTimer(runAt.Sub(curTime))
		zap.L().Debug("Scheduling task", zap.String("task", t.Name()), zap.String("runAt", runAt.Format(time.RFC3339)))

		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case <-ctx.Done():
				timer.Stop()
				zap.L().Debug("Cancelling scheduled (but not running) task, orchestrator context is cancelled.", zap.String("task", t.Name()))
				return
			case <-timer.C:
				lock.Lock()
				defer lock.Unlock()
				zap.L().Debug("Running task", zap.String("task", t.Name()))
				
				// Task execution runs with mainCtx meaning config changes do not interrupt it, but cancelling the orchestration loop will.
				if err := t.Run(mainCtx); err != nil {
					zap.L().Error("Task failed", zap.String("task", t.Name()), zap.Error(err))
				} else {
					zap.L().Debug("Task finished", zap.String("task", t.Name()))
				}

				if ctx.Err() != nil {
					zap.L().Debug("Not attempting to reschedule task, orchestrator context is cancelled.", zap.String("task", t.Name()))
					return 
				}

				execTask(t)
			}
		}()
	}

	// Schedule all backup tasks.
	for _, plan := range config.Plans {
		t, err := NewScheduledBackupTask(o, plan)
		if err != nil {
			zap.L().Error("Failed to create backup task for plan", zap.String("plan", plan.Id), zap.Error(err))
		}

		execTask(t)
	}

	// wait for either an error or the context to be cancelled, then wait for all tasks.
	for {
		select {
		case t := <-o.externTasks:
			execTask(t)
		case <-ctx.Done():
			cancel()
			wg.Wait()
			return 
		case <-o.configUpdates:
			zap.L().Info("Orchestrator received config change, waiting for in-progress operations")
			cancel()
			wg.Wait()
			return
		}
	}
}

func (o *Orchestrator) EnqueueTask(t Task) {
	o.externTasks <- t
}

// resticRepoPool caches restic repos.
type resticRepoPool struct {
	mu sync.Mutex
	repos map[string]*RepoOrchestrator
	configProvider config.ConfigStore
}


func newResticRepoPool(configProvider config.ConfigStore) *resticRepoPool {
	return &resticRepoPool{
		repos: make(map[string]*RepoOrchestrator),
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
	delete(rp.repos, repoId);

	var opts []restic.GenericOption
	opts = append(opts, restic.WithPropagatedEnvVars(restic.EnvToPropagate...))
	if len(repoProto.GetEnv()) > 0 {
		opts = append(opts, restic.WithEnv(repoProto.GetEnv()...))
	}
	if len(repoProto.GetFlags()) > 0 {
		opts = append(opts, restic.WithFlags(repoProto.GetFlags()...))
	}

	// Otherwise create a new repo.
	repo = newRepoOrchestrator(repoProto, restic.NewRepo(repoProto, opts...))
	rp.repos[repoId] = repo
	return repo, nil
}
