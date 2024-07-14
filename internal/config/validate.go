package config

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/config/validationutil"
	"github.com/garethgeorge/backrest/internal/env"
	"github.com/garethgeorge/backrest/internal/protoutil"
	"github.com/hashicorp/go-multierror"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

var HubInstanceID = "_hub_" // instance ID for the hub server

// remoteUriRegex matches URIs that are not local paths.
var remoteUriRegex = regexp.MustCompile(`^[a-zA-Z0-9]+:.*$`)

func ValidateConfig(c *v1.Config) error {
	var err error

	if env.IsHubServer() {
		// The hub server must have the instance ID set to _hub_ to differentiate it from a daemon.
		if c.Instance == "" {
			c.Instance = HubInstanceID
		} else if c.Instance != HubInstanceID {
			err = multierror.Append(err, fmt.Errorf("hub server instance ID must be %q, is this backrest install already initialized as a daemon?", HubInstanceID))
		}
	} else {
		if e := validationutil.ValidateID(c.Instance, validationutil.IDMaxLen); e != nil {
			if errors.Is(e, validationutil.ErrEmpty) {
				zap.L().Warn("ACTION REQUIRED: instance ID is empty, will be required in a future update. Please open the backrest UI to set a unique instance ID. Until fixed this warning (and related errors) will print periodically.")
			} else {
				err = multierror.Append(err, fmt.Errorf("instance ID %q invalid: %w", c.Instance, e))
			}
		}
	}

	repos := make(map[string]*v1.Repo)
	if c.Repos != nil {
		for _, repo := range c.Repos {
			if e := validateRepo(repo); e != nil {
				err = multierror.Append(err, fmt.Errorf("repo %s: %w", repo.GetId(), e))
			}
			if _, ok := repos[repo.Id]; ok {
				err = multierror.Append(err, fmt.Errorf("repo %s: duplicate id", repo.GetId()))
			}
			repos[repo.Id] = repo
		}
		slices.SortFunc(c.Repos, func(a, b *v1.Repo) int {
			if a.Id < b.Id {
				return -1
			}
			return 1
		})
	}

	if c.Plans != nil {
		plans := make(map[string]*v1.Plan)
		for _, plan := range c.Plans {
			if _, ok := plans[plan.Id]; ok {
				err = multierror.Append(err, fmt.Errorf("plan %s: duplicate id", plan.GetId()))
			}
			plans[plan.Id] = plan
			if e := validatePlan(plan, repos); e != nil {
				err = multierror.Append(err, fmt.Errorf("plan %s: %w", plan.GetId(), e))
			}
		}
		slices.SortFunc(c.Plans, func(a, b *v1.Plan) int {
			if a.Id < b.Id {
				return -1
			}
			return 1
		})
	}

	return err
}

func validateRepo(repo *v1.Repo) error {
	var err error
	if e := validationutil.ValidateID(repo.Id, 0); e != nil {
		err = multierror.Append(err, fmt.Errorf("id %q invalid: %w", repo.Id, e))
	}

	if repo.Uri == "" {
		err = multierror.Append(err, errors.New("uri is required"))
	} else if env.IsHubServer() && !remoteUriRegex.MatchString(repo.Uri) {
		err = multierror.Append(err, fmt.Errorf("uri %q must be a remote repo, local storage is not supported for hubs", repo.Uri))
	}

	if repo.PrunePolicy.GetSchedule() != nil {
		if e := protoutil.ValidateSchedule(repo.PrunePolicy.GetSchedule()); e != nil {
			err = multierror.Append(err, fmt.Errorf("prune policy schedule: %w", e))
		}
	}

	if repo.CheckPolicy.GetSchedule() != nil {
		if e := protoutil.ValidateSchedule(repo.CheckPolicy.GetSchedule()); e != nil {
			err = multierror.Append(err, fmt.Errorf("check policy schedule: %w", e))
		}
	}

	for _, env := range repo.Env {
		if !strings.Contains(env, "=") {
			err = multierror.Append(err, fmt.Errorf("invalid env var %s, must take format KEY=VALUE", env))
		}
	}

	slices.Sort(repo.Env)

	return err
}

func validatePlan(plan *v1.Plan, repos map[string]*v1.Repo) error {
	var err error
	if e := validationutil.ValidateID(plan.Id, 0); e != nil {
		err = multierror.Append(err, fmt.Errorf("id %q invalid: %w", plan.Id, e))
	}

	if plan.Schedule != nil {
		if e := protoutil.ValidateSchedule(plan.Schedule); e != nil {
			err = multierror.Append(err, fmt.Errorf("schedule: %w", e))
		}
	}

	for idx, p := range plan.Paths {
		if p == "" {
			err = multierror.Append(err, fmt.Errorf("path[%d] cannot be empty", idx))
		}
	}

	if plan.Repo == "" {
		err = multierror.Append(err, fmt.Errorf("repo is required"))
	}

	if _, ok := repos[plan.Repo]; !ok {
		err = multierror.Append(err, fmt.Errorf("repo %q not found", plan.Repo))
	}

	if plan.Retention != nil && plan.Retention.Policy == nil {
		err = multierror.Append(err, errors.New("retention policy must be nil or must specify a policy"))
	} else if policyTimeBucketed, ok := plan.Retention.GetPolicy().(*v1.RetentionPolicy_PolicyTimeBucketed); ok {
		if proto.Equal(policyTimeBucketed.PolicyTimeBucketed, &v1.RetentionPolicy_TimeBucketedCounts{}) {
			err = multierror.Append(err, errors.New("time bucketed policy must specify a non-empty bucket"))
		}
	}

	slices.Sort(plan.Paths)

	return err
}
