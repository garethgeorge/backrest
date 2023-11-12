package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	v1 "github.com/garethgeorge/resticui/gen/go/v1"
	"github.com/google/renameio"
	"google.golang.org/protobuf/encoding/protojson"
	yaml "gopkg.in/yaml.v3"
)

type YamlFileStore struct {
	Path string
	mu sync.Mutex
}

var _ ConfigStore = &YamlFileStore{}

func (f *YamlFileStore) Get() (*v1.Config, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	data, err := os.ReadFile(f.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrConfigNotFound
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}


	data, err = yamlToJson(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	var config v1.Config
	
	if err = protojson.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := ValidateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &config, nil
}

func (f *YamlFileStore) Update(config *v1.Config) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if err := ValidateConfig(config); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	data, err := protojson.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	data, err = jsonToYaml(data)
	if err != nil {
		return fmt.Errorf("failed to convert config to yaml: %w", err)
	}

	err = os.MkdirAll(filepath.Dir(f.Path), 0755)
	if err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	err = renameio.WriteFile(f.Path, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func jsonToYaml(data []byte) ([]byte, error) {
	var config interface{}
	err := json.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return yaml.Marshal(config)
}

func yamlToJson(data []byte) ([]byte, error) {
	var config interface{}
	err := yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return json.Marshal(config)
}