package config

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/config/stringutil"
	"github.com/gitploy-io/cronexpr"
	"github.com/hashicorp/go-multierror"
)

func ValidateConfig(c *v1.Config) error {
	var err error

	c.Instance, err = validateID(c.Instance)
	if err != nil {
		err = multierror.Append(err, fmt.Errorf("instance ID: %w", err))
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

	if repo.Id == "" || !stringutil.ValidateID(repo.Id) {
		err = multierror.Append(err, fmt.Errorf("id %q contains invalid characters (or empty)", repo.Id))
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
	if plan.Paths == nil || len(plan.Paths) == 0 {
		err = multierror.Append(err, fmt.Errorf("path is required"))
	}

	if plan.Id == "" || !stringutil.ValidateID(plan.Id) {
		err = multierror.Append(err, fmt.Errorf("id %q contains invalid characters (or empty)", plan.Id))
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
	}

	slices.Sort(plan.Paths)
	slices.Sort(plan.Excludes)
	slices.Sort(plan.Iexcludes)

	return err
}

func validateID(id string) (string, error) {
	if len(id) > 32 {
		return "", fmt.Errorf("id %q is too long", id)
	}
	return stringutil.SanitizeID(id), nil
}
