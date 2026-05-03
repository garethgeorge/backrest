package config

import (
	"sync"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"google.golang.org/protobuf/proto"
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

func (c *MemoryStore) Transform(fn func(cfg *v1.Config) (*v1.Config, error)) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	cloned := proto.Clone(c.Config).(*v1.Config)
	result, err := fn(cloned)
	if err != nil {
		return err
	}
	if result != nil {
		c.Config = result
	}
	return nil
}
