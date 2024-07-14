package config

import (
	"slices"
	"sync"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
)

type ConfigManager struct {
	// ConfigStore is the store for the config
	ConfigStore ConfigStore

	mu       sync.RWMutex
	onChange []*func(config *v1.Config)
}

var _ ConfigStore = &ConfigManager{}

func (c *ConfigManager) Get() (*v1.Config, error) {
	return c.ConfigStore.Get()
}

func (c *ConfigManager) Update(config *v1.Config) error {
	err := c.ConfigStore.Update(config)
	if err != nil {
		return err
	}
	c.notifyChange(config)
	return nil
}

func (c *ConfigManager) Subscribe(f func(config *v1.Config)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onChange = append(c.onChange, &f)
}

func (c *ConfigManager) Unsubscribe(f func(config *v1.Config)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onChange = slices.DeleteFunc(c.onChange, func(h *func(config *v1.Config)) bool {
		return h == &f
	})
}

func (c *ConfigManager) notifyChange(config *v1.Config) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, f := range c.onChange {
		(*f)(config)
	}
}
