package config

import (
	"sync"

	v1 "github.com/garethgeorge/restora/gen/go/v1"
)

type MemoryStore struct {
	mu     sync.Mutex
	Config *v1.Config
}

var _ ConfigStore = &MemoryStore{}

func (c *MemoryStore) Get() (*v1.Config, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Config, nil
}

func (c *MemoryStore) Update(config *v1.Config) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Config = config
	return nil
}
