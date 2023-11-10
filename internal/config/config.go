package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path"

	v1 "github.com/garethgeorge/resticui/gen/go/v1"
	multierror "github.com/hashicorp/go-multierror"
)

var configDirFlag = flag.String("config_dir", "", "The directory to store the config file")

var Default ConfigStore = &YamlFileStore{
	Path: path.Join(configDir(*configDirFlag), "config.yaml"),
}

type ConfigStore interface {
	Get() (*v1.Config, error)
	Update(config *v1.Config) error
}

func NewDefaultConfig() *v1.Config {
	return &v1.Config{
		LogDir: "/var/log/resticui",
		Repos: []*v1.Repo{},
		Plans: []*v1.Plan{},
	}
}

func ValidateConfig(c *v1.Config) error {
	if c.LogDir == "" {
		return errors.New("log_dir is required")
	}

	if c.Repos == nil {
		return errors.New("repos is required")
	}

	if c.Plans == nil {
		return errors.New("plans is required")
	}

	var error error

	repos := make(map[string]*v1.Repo)
	for _, repo := range c.Repos {
		if repo.GetId() == "" {
			error = multierror.Append(error, fmt.Errorf("repo name is required"))
		}
		repos[repo.GetId()] = repo
	}

	for _, plan := range c.Plans {
		if plan.Paths == nil || len(plan.Paths) == 0 {
			error = multierror.Append(error, fmt.Errorf("plan %s: path is required", plan.GetId()))
		}

		if plan.Repo == "" {
			error = multierror.Append(error,fmt.Errorf("plan %s: repo is required", plan.GetId()))
		}

		if _, ok := repos[plan.Repo]; !ok {
			error = multierror.Append(error, fmt.Errorf("plan %s: repo %s not found", plan.GetId(), plan.Repo))
		}
	}

	return error
}

func configDir(override string) string {
	if override != "" {
		return override
	}

	if env := os.Getenv("XDG_CONFIG_HOME"); env != "" {
		return path.Join(env, "resticui")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	return fmt.Sprintf("%v/.config/resticui", home)
}