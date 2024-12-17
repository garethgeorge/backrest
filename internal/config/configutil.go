package config

import (
	v1 "github.com/garethgeorge/backrest/gen/go/v1"
)

func FindPlan(cfg *v1.Config, planID string) *v1.Plan {
	for _, plan := range cfg.Plans {
		if plan.Id == planID {
			return plan
		}
	}
	return nil
}

func FindRepo(cfg *v1.Config, repoID string) *v1.Repo {
	for _, repo := range cfg.Repos {
		if repo.Id == repoID {
			return repo
		}
	}
	return nil
}
