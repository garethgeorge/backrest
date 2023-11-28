package config

import (
	"errors"
	"fmt"
	"os"
	"path"
	"sync"

	v1 "github.com/garethgeorge/resticui/gen/go/v1"
)

var ErrConfigNotFound = fmt.Errorf("config not found")

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

type CachingValidatingStore struct {
	ConfigStore
	mu     sync.Mutex
	config *v1.Config
}

func (c *CachingValidatingStore) Get() (*v1.Config, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.config != nil {
		return c.config, nil
	}

	config, err := c.ConfigStore.Get()
	if err != nil {
		if errors.Is(err, ErrConfigNotFound) {
			c.config = NewDefaultConfig()
			return c.config, nil
		}
		return c.config, err
	}

	if err := ValidateConfig(config); err != nil {
		return nil, err
	}

	c.config = config
	return config, nil
}

func (c *CachingValidatingStore) Update(config *v1.Config) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := ValidateConfig(config); err != nil {
		return err
	}

	if err := c.ConfigStore.Update(config); err != nil {
		return err
	}

	c.config = config
	return nil
}
