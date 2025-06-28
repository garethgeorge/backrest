package syncapi

import (
	"errors"
	"fmt"
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

	ErrRemoteConfigNotFound = errors.New("remote config not found")
)

type RemoteConfigStore interface {
	// Get a remote config for the given instance ID.
	Get(peer *v1.Multihost_Peer) (*v1.RemoteConfig, error)
	// Update or create a remote config for the given instance ID.
	Update(peer *v1.Multihost_Peer, config *v1.RemoteConfig) error
	// Delete a remote config for the given instance ID.
	Delete(peer *v1.Multihost_Peer) error
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

func (s *jsonDirRemoteConfigStore) Get(peer *v1.Multihost_Peer) (*v1.RemoteConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if peer == nil || peer.InstanceId == "" || peer.Keyid == "" {
		return nil, errors.New("peer and peer.InstanceId and peer.Keyid are required")
	}

	if config, ok := s.cache[peer.Keyid]; ok {
		return config, nil
	}

	file := s.fileForInstance(peer.InstanceId, peer.Keyid)
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

	s.cache[peer.Keyid] = &config
	return &config, nil
}

func (s *jsonDirRemoteConfigStore) Update(peer *v1.Multihost_Peer, config *v1.RemoteConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if peer == nil || peer.InstanceId == "" || peer.Keyid == "" {
		return errors.New("peer and peer.InstanceId and peer.Keyid are required")
	}

	file := s.fileForInstance(peer.InstanceId, peer.Keyid)
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

	s.cache[peer.Keyid] = config
	return nil
}

func (s *jsonDirRemoteConfigStore) Delete(peer *v1.Multihost_Peer) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if peer == nil || peer.InstanceId == "" || peer.Keyid == "" {
		return errors.New("peer and peer.InstanceId and peer.Keyid are required")
	}

	file := s.fileForInstance(peer.InstanceId, peer.Keyid)
	if err := os.Remove(file); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove config file: %w", err)
	}

	delete(s.cache, peer.Keyid)
	return nil
}

func (s *jsonDirRemoteConfigStore) fileForInstance(instanceID string, keyID string) string {
	safeInstanceID := strings.Replace(instanceID, "..", ".", -1)
	safeInstanceID = sanitizeFilenameRegex.ReplaceAllString(safeInstanceID, "_")
	return filepath.Join(s.dir, fmt.Sprintf("%s-%s.json", safeInstanceID, keyID))
}

type memoryConfigStore struct {
	mu      sync.Mutex
	configs map[string]*v1.RemoteConfig
}

func newMemoryConfigStore() RemoteConfigStore {
	return &memoryConfigStore{
		configs: make(map[string]*v1.RemoteConfig),
	}
}

func (s *memoryConfigStore) Get(peer *v1.Multihost_Peer) (*v1.RemoteConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if peer == nil || peer.InstanceId == "" || peer.Keyid == "" {
		return nil, errors.New("peer and peer.InstanceId and peer.Keyid are required")
	}

	if config, ok := s.configs[peer.Keyid]; ok {
		return config, nil
	}
	return nil, ErrRemoteConfigNotFound
}

func (s *memoryConfigStore) Update(peer *v1.Multihost_Peer, config *v1.RemoteConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if peer == nil || peer.InstanceId == "" || peer.Keyid == "" {
		return errors.New("peer and peer.InstanceId and peer.Keyid are required")
	}

	s.configs[peer.Keyid] = config
	return nil
}

func (s *memoryConfigStore) Delete(peer *v1.Multihost_Peer) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if peer == nil || peer.InstanceId == "" || peer.Keyid == "" {
		return errors.New("peer and peer.InstanceId and peer.Keyid are required")
	}

	delete(s.configs, peer.Keyid)
	return nil
}

func GetRepoConfig(store RemoteConfigStore, peer *v1.Multihost_Peer, repoID string) (*v1.Repo, error) {
	config, err := store.Get(peer)
	if err != nil {
		return nil, fmt.Errorf("get %q (%q): %w", peer.InstanceId, peer.Keyid, err)
	}
	for _, repo := range config.Repos {
		if repo.Id == repoID {
			return repo, nil
		}
	}
	return nil, fmt.Errorf("get %q/%q: %w", peer.InstanceId, repoID, ErrRemoteConfigNotFound)
}
