package config

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/config/validationutil"
	"github.com/gitploy-io/cronexpr"
	"github.com/hashicorp/go-multierror"
	"google.golang.org/protobuf/proto"
)

func ValidateConfig(c *v1.Config) error {
	var err error

	if e := validationutil.ValidateID(c.Instance, validationutil.IDMaxLen); e != nil {
		err = multierror.Append(err, fmt.Errorf("instance ID %q invalid: %w", c.Instance, e))
	}

	repos := make(map[string]*v1.Repo)
	if c.Repos != nil {
		for _, repo := range c.Repos {
			if e := validateRepo(repo); e != nil {
				err = multierror.Append(e, fmt.Errorf("repo %s: %w", repo.GetId(), err))
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

	if _, e := cronexpr.Parse(plan.Cron); e != nil {
		err = multierror.Append(err, fmt.Errorf("invalid cron %q: %w", plan.Cron, e))
	}

	if plan.Retention != nil && plan.Retention.Policy == nil {
		err = multierror.Append(err, errors.New("retention policy must be nil or must specify a policy"))
	} else if policyTimeBucketed, ok := plan.Retention.Policy.(*v1.RetentionPolicy_PolicyTimeBucketed); ok {
		if proto.Equal(policyTimeBucketed.PolicyTimeBucketed, &v1.RetentionPolicy_TimeBucketedCounts{}) {
			err = multierror.Append(err, errors.New("time bucketed policy must specify a non-empty bucket"))
		}
	}

	slices.Sort(plan.Paths)
	slices.Sort(plan.Excludes)
	slices.Sort(plan.Iexcludes)

	return err
}
