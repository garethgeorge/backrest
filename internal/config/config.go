package config

import (
	"errors"
	"fmt"
	"os"
	"path"
	"sync"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/config/migrations"
	"go.uber.org/zap"
)

var ErrConfigNotFound = fmt.Errorf("config not found")

type ConfigStore interface {
	Get() (*v1.Config, error)
	Update(config *v1.Config) error
}

func NewDefaultConfig() *v1.Config {
	hostname, _ := os.Hostname()
	return &v1.Config{
		Host:  hostname,
		Repos: []*v1.Repo{},
		Plans: []*v1.Plan{},
	}
}

func configDir(override string) string {
	if override != "" {
		return override
	}

	if env := os.Getenv("XDG_CONFIG_HOME"); env != "" {
		return path.Join(env, "backrest")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	return fmt.Sprintf("%v/.config/backrest", home)
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

	// Check if we need to migrate
	if config.Version < migrations.CurrentVersion {
		zap.S().Infof("Migrating config from version %d to %d", config.Version, migrations.CurrentVersion)
		if err := migrations.ApplyMigrations(config); err != nil {
			return nil, err
		}

		if config.Version != migrations.CurrentVersion {
			return nil, fmt.Errorf("migration failed to update config to version %d", migrations.CurrentVersion)
		}

		// Write back the migrated config.
		if err := c.ConfigStore.Update(config); err != nil {
			return nil, err
		}
	}

	// Validate the config
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
