package config

import (
	"flag"
	"fmt"
	"os"
	"path"

	v1 "github.com/garethgeorge/resticui/gen/go/v1"
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
		Repos: []*v1.Repo{},
		Plans: []*v1.Plan{},
	}
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