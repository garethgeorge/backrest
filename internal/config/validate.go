package config

import (
	"errors"
	"fmt"
	"strings"

	v1 "github.com/garethgeorge/resticui/gen/go/v1"
	"github.com/hashicorp/go-multierror"
)

func validateConfig(c *v1.Config) error {
	if c.LogDir == "" {
		return errors.New("log_dir is required")
	}

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
			if plan.Paths == nil || len(plan.Paths) == 0 {
				err = multierror.Append(err, fmt.Errorf("plan %s: path is required", plan.GetId()))
			}

			if plan.Repo == "" {
				err = multierror.Append(err,fmt.Errorf("plan %s: repo is required", plan.GetId()))
			}

			if _, ok := repos[plan.Repo]; !ok {
				err = multierror.Append(err, fmt.Errorf("plan %s: repo %s not found", plan.GetId(), plan.Repo))
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