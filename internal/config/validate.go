package config

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	v1 "github.com/garethgeorge/resticui/gen/go/v1"
	"github.com/gitploy-io/cronexpr"
	"github.com/hashicorp/go-multierror"
)

func ValidateConfig(c *v1.Config) error {
	var err error
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
	} 

	return err
}

func validateRepo(repo *v1.Repo) error {
	var err error
	if repo.Id == "" {
		err = multierror.Append(err, errors.New("id is required"))
	}

	if repo.Uri == "" {
		err = multierror.Append(err, errors.New("uri is required"))
	}

	if repo.Password == "" {
		err = multierror.Append(err, errors.New("password is required"))
	}

	for _, env := range repo.Env {
		if !strings.Contains(env, "=") {
			err = multierror.Append(err, fmt.Errorf("invalid env var %s, must take format KEY=VALUE", env))
		}
	}

	return err
}

func validatePlan(plan *v1.Plan, repos map[string]*v1.Repo) error {
	var err error
	if plan.Paths == nil || len(plan.Paths) == 0 {
		err = multierror.Append(err, fmt.Errorf("path is required"))
	}

	for idx, p := range plan.Paths {
		if p == "" {
			err = multierror.Append(err, fmt.Errorf("path[%d] cannot be empty", idx))
		}
	}

	if plan.Repo == "" {
		err = multierror.Append(err,fmt.Errorf("repo is required"))
	}

	if _, ok := repos[plan.Repo]; !ok {
		err = multierror.Append(err, fmt.Errorf("repo %q not found", plan.Repo))
	}

	
	if _, e := cronexpr.Parse(plan.Cron); e != nil {
		err = multierror.Append(err, fmt.Errorf("invalid cron %q: %w", plan.Cron, e))
	}

	if plan.GetRetention() != nil {
		if e := validateRetention(plan.Retention); e != nil {
			err = multierror.Append(err, fmt.Errorf("invalid retention policy: %w", e))
		}
	}

	return err
}

func validateRetention(policy *v1.RetentionPolicy) error {
	var err error
	if policy.KeepWithinDuration != "" {
		match, e := regexp.Match(`(\d+h)?(\d+m)?(\d+s)?`, []byte(policy.KeepWithinDuration))
		if e != nil {
			panic(e) // regex error
		}
		if !match {
			err = multierror.Append(err, fmt.Errorf("invalid keep_within_duration %q", policy.KeepWithinDuration))
		}
		if policy.KeepLastN != 0 || policy.KeepHourly != 0 || policy.KeepDaily != 0 || policy.KeepWeekly != 0 || policy.KeepMonthly != 0 || policy.KeepYearly != 0 {
			err = multierror.Append(err, fmt.Errorf("keep_within_duration cannot be used with other retention settings"))
		}
	} else {
		if policy.KeepLastN == 0 && policy.KeepHourly == 0 && policy.KeepDaily == 0 && policy.KeepWeekly == 0 && policy.KeepMonthly == 0 && policy.KeepYearly == 0 {
			err = multierror.Append(err, fmt.Errorf("at least one retention policy must be set"))
		}
	}
	return err
}