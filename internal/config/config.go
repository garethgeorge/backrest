package config

import (
	"errors"
	"fmt"
	"sync"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/config/migrations"
	"github.com/garethgeorge/backrest/internal/cryptoutil"
	"github.com/garethgeorge/backrest/internal/eventemitter"
	"go.uber.org/zap"
)

var ErrConfigNotFound = fmt.Errorf("config not found")

type ConfigManager struct {
	Store    ConfigStore
	OnChange eventemitter.EventEmitter[struct{}]
}

var _ ConfigStore = &ConfigManager{}

func (m *ConfigManager) Get() (*v1.Config, error) {
	return m.Store.Get()
}

func (m *ConfigManager) Update(config *v1.Config) error {
	err := m.Store.Update(config)
	if err != nil {
		return err
	}
	m.OnChange.Emit(struct{}{})
	return nil
}

type ConfigStore interface {
	Get() (*v1.Config, error)
	Update(config *v1.Config) error
}

func NewDefaultConfig() *v1.Config {
	cfg := &v1.Config{
		Version:  migrations.CurrentVersion,
		Instance: "",
		Repos:    []*v1.Repo{},
		Plans:    []*v1.Plan{},
		Auth: &v1.Auth{
			Disabled: true,
		},
	}
	_, err := PopulateRequiredFields(cfg)
	if err != nil {
		zap.S().Fatalf("failed to populate required fields in default config: %v", err)
	}
	return cfg
}

func PopulateRequiredFields(config *v1.Config) (mutated bool, err error) {
	if config.GetMultihost() == nil {
		config.Multihost = &v1.Multihost{}
		mutated = true
	}
	if config.GetMultihost().Identity == nil {
		identity, err := cryptoutil.GeneratePrivateKey()
		if err != nil {
			return false, fmt.Errorf("generate private key: %w", err)
		}
		config.GetMultihost().Identity = identity
		mutated = true
	}
	return
}

// TODO: merge caching validating store functions into config manager
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
	mutated, err := PopulateRequiredFields(config)
	if err != nil {
		return nil, fmt.Errorf("populate required fields: %w", err)
	}
	if config.Version < migrations.CurrentVersion {
		zap.S().Infof("migrating config from version %d to %d", config.Version, migrations.CurrentVersion)
		if err := migrations.ApplyMigrations(config); err != nil {
			return nil, err
		}
		mutated = true
	}
	if mutated {
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
