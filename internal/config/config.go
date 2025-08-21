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

	migrateOnce sync.Once
	migrateErr  error

	cachedMu sync.Mutex
	cached   *v1.Config
}

var _ ConfigStore = &ConfigManager{}

func (m *ConfigManager) migrate(config *v1.Config) error {
	// Check if we need to migrate
	mutated, err := PopulateRequiredFields(config)
	if err != nil {
		return fmt.Errorf("populate required fields: %w", err)
	}
	if config.Version < migrations.CurrentVersion {
		zap.S().Infof("migrating config from version %d to %d", config.Version, migrations.CurrentVersion)
		if err := migrations.ApplyMigrations(config); err != nil {
			return err
		}
		mutated = true
	}
	if mutated {
		// Check validations
		if err := ValidateConfig(config); err != nil {
			return fmt.Errorf("validation after migration: %v", err)
		}

		// Write back the migrated config.
		if err := m.Store.Update(config); err != nil {
			return err
		}
	}
	return nil
}

func (m *ConfigManager) Get() (*v1.Config, error) {
	m.cachedMu.Lock()
	defer m.cachedMu.Unlock()

	if m.cached != nil {
		return m.cached, nil
	}

	config, err := m.Store.Get()
	if err != nil {
		if errors.Is(err, ErrConfigNotFound) {
			m.cached = NewDefaultConfig()
			return m.cached, nil
		}
		return nil, err
	}

	// Try to apply migrations
	m.migrateOnce.Do(func() {
		m.migrateErr = m.migrate(config)
	})
	if m.migrateErr != nil {
		return nil, m.migrateErr
	}

	// Validate the config
	if err := ValidateConfig(config); err != nil {
		return nil, err
	}

	// Finally cache it for performance
	m.cached = config

	return config, err
}

func (m *ConfigManager) Update(config *v1.Config) error {
	m.cachedMu.Lock()
	defer m.cachedMu.Unlock()

	if err := ValidateConfig(config); err != nil {
		return err
	}

	err := m.Store.Update(config)
	if err != nil {
		return err
	}
	m.cached = config
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
