package syncapi

import (
	"errors"
	"fmt"
	"hash/crc32"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

var (
	sanitizeFilenameRegex = regexp.MustCompile("[^a-zA-Z0-9\\-_\\.]+")

	ErrRemoteConfigNotFound = errors.New("config for remote instance not found")
)

type RemoteConfigStore interface {
	// Get a remote config for the given instance ID.
	Get(instanceID string) (*v1.RemoteConfig, error)
	// Update or create a remote config for the given instance ID.
	Update(instanceID string, config *v1.RemoteConfig) error
	// Delete a remote config for the given instance ID.
	Delete(instanceID string) error
}

type jsonDirRemoteConfigStore struct {
	mu    sync.Mutex
	dir   string
	cache map[string]*v1.RemoteConfig
}

func NewJSONDirRemoteConfigStore(dir string) RemoteConfigStore {
	return &jsonDirRemoteConfigStore{
		dir:   dir,
		cache: make(map[string]*v1.RemoteConfig),
	}
}

func (s *jsonDirRemoteConfigStore) Get(instanceID string) (*v1.RemoteConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if instanceID == "" {
		return nil, errors.New("instanceID is required")
	}

	if config, ok := s.cache[instanceID]; ok {
		return config, nil
	}

	file := s.fileForInstance(instanceID)
	data, err := os.ReadFile(file)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrRemoteConfigNotFound
		}
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var config v1.RemoteConfig
	if err = (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	s.cache[instanceID] = &config
	return &config, nil
}

func (s *jsonDirRemoteConfigStore) Update(instanceID string, config *v1.RemoteConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if instanceID == "" {
		return errors.New("instanceID is required")
	}

	file := s.fileForInstance(instanceID)
	data, err := protojson.MarshalOptions{
		Indent:    "  ",
		Multiline: true,
	}.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	err = os.MkdirAll(filepath.Dir(file), 0755)
	if err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	err = os.WriteFile(file, data, 0600)
	if err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	s.cache[instanceID] = config
	return nil
}

func (s *jsonDirRemoteConfigStore) Delete(instanceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if instanceID == "" {
		return errors.New("instanceID is required")
	}

	file := s.fileForInstance(instanceID)
	if err := os.Remove(file); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove config file: %w", err)
	}

	delete(s.cache, instanceID)
	return nil
}

func (s *jsonDirRemoteConfigStore) fileForInstance(instanceID string) string {
	safeInstanceID := strings.Replace(instanceID, "..", ".", -1)
	safeInstanceID = sanitizeFilenameRegex.ReplaceAllString(safeInstanceID, "_")
	checksum := crc32.ChecksumIEEE([]byte(instanceID)) // checksum eliminates collisions in the case of replacing characters.
	return filepath.Join(s.dir, fmt.Sprintf("%s-%08x.json", safeInstanceID, checksum))
}

type memoryConfigStore struct {
	configs map[string]*v1.RemoteConfig
}

func newMemoryConfigStore() *memoryConfigStore {
	return &memoryConfigStore{
		configs: make(map[string]*v1.RemoteConfig),
	}
}

func (s *memoryConfigStore) Get(instanceID string) (*v1.RemoteConfig, error) {
	if config, ok := s.configs[instanceID]; ok {
		return config, nil
	}
	return nil, ErrRemoteConfigNotFound
}

func (s *memoryConfigStore) Update(instanceID string, config *v1.RemoteConfig) error {
	if instanceID == "" {
		return errors.New("instanceID is required")
	}
	s.configs[instanceID] = config
	return nil
}

func (s *memoryConfigStore) Delete(instanceID string) error {
	if instanceID == "" {
		return errors.New("instanceID is required")
	}
	delete(s.configs, instanceID)
	return nil
}
