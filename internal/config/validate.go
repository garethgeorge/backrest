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

func validateConfig(c *v1.Config) error {
	var err error
	repos := make(map[string]*v1.Repo)
	if c.Repos != nil {
		for _, repo := range c.Repos {
			if e := validateRepo(repo); e != nil {
				err = multierror.Append(e, fmt.Errorf("repo %s: %w", repo.GetId(), err))
			}
			repos[repo.GetId()] = repo
		}
	}

	if c.Plans != nil {
		for _, plan := range c.Plans {
			err := validatePlan(plan, repos);
			if err != nil {
				err = multierror.Append(err, fmt.Errorf("plan %s: %w", plan.GetId(), err))
			}
		}
	} 

	return err
}

func validateRepo(repo *v1.Repo) error {
	var err error
	if repo.GetId() == "" {
		err = multierror.Append(err, errors.New("id is required"))
	}

	if repo.GetUri() == "" {
		err = multierror.Append(err, errors.New("uri is required"))
	}

	if repo.GetPassword() == "" {
		err = multierror.Append(err, errors.New("password is required"))
	}

	for _, env := range repo.GetEnv() {
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

	if plan.Repo == "" {
		err = multierror.Append(err,fmt.Errorf("repo is required"))
	}

	if _, ok := repos[plan.Repo]; !ok {
		err = multierror.Append(err, fmt.Errorf("repo %q not found", plan.Repo))
	}

	
	if _, e := cronexpr.Parse(plan.GetCron()); err != nil {
		err = multierror.Append(err, fmt.Errorf("invalid cron %q: %w", plan.GetCron(), e))
	}

	if plan.GetRetention() != nil {
		if e := validateRetention(plan.GetRetention()); e != nil {
			err = multierror.Append(err, fmt.Errorf("invalid retention policy: %w", e))
		}
	}

	return err
}

func validateRetention(policy *v1.RetentionPolicy) error {
	var err error
	if policy.GetKeepWithinDuration() != "" {
		match, e := regexp.Match(`(\d+h)?(\d+m)?(\d+s)?`, []byte(policy.GetKeepWithinDuration()))
		if e != nil {
			panic(e) // regex error
		}
		if !match {
			err = multierror.Append(err, fmt.Errorf("invalid keep_within_duration %q", policy.GetKeepWithinDuration()))
		}

		if policy.GetKeepLastN() != 0 || policy.GetKeepHourly() != 0 || policy.GetKeepDaily() != 0 || policy.GetKeepWeekly() != 0 || policy.GetKeepMonthly() != 0 || policy.GetKeepYearly() != 0 {
			err = multierror.Append(err, fmt.Errorf("keep_within_duration cannot be used with other retention settings"))
		}
	} else {
		if policy.GetKeepLastN() == 0 && policy.GetKeepHourly() == 0 && policy.GetKeepDaily() == 0 && policy.GetKeepWeekly() == 0 && policy.GetKeepMonthly() == 0 && policy.GetKeepYearly() == 0 {
			err = multierror.Append(err, fmt.Errorf("at least one retention policy must be set"))
		}
	}
	return err
}