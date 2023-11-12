package orchestrator

import (
	"errors"
	"fmt"
	"sync"

	v1 "github.com/garethgeorge/resticui/gen/go/v1"
	"github.com/garethgeorge/resticui/internal/config"
	"github.com/garethgeorge/resticui/pkg/restic"
	"google.golang.org/protobuf/proto"
)

var ErrRepoNotFound = errors.New("repo not found")
var ErrRepoInitializationFailed = errors.New("repo initialization failed")
var ErrPlanNotFound = errors.New("plan not found")

// Orchestrator is responsible for managing repos and backups.
type Orchestrator struct {
	configProvider config.ConfigStore
	repoPool *resticRepoPool
}

func NewOrchestrator(configProvider config.ConfigStore) *Orchestrator {
	return &Orchestrator{
		configProvider: configProvider,
		repoPool: newResticRepoPool(configProvider),
	}
}

func (o *Orchestrator) GetRepo(repoId string) (repo *RepoOrchestrator, err error) {
	r, err := o.repoPool.GetRepo(repoId)
	if  err != nil {
		return nil, fmt.Errorf("failed to get repo %q: %w", repoId, err)
	}
	return r, nil
}

func (o *Orchestrator) GetPlan(planId string) (*v1.Plan, error) {
	cfg, err := o.configProvider.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	if cfg.Plans == nil {
		return nil, ErrPlanNotFound
	}

	for _, p := range cfg.Plans {
		if p.Id == planId {
			return p, nil
		}
	}

	return nil, ErrPlanNotFound
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
