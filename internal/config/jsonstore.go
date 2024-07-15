package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/natefinch/atomic"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
)

var (
	configKeepVersions = 10
)

type JsonFileStore[T any] struct {
	Path string
	mu   sync.Mutex
}

var _ ConfigStore = &JsonFileStore[any]{}

func (f *JsonFileStore[T]) Get() (*v1.Config, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	data, err := os.ReadFile(f.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrConfigNotFound
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config v1.Config
	if strictErr := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(data, &config); strictErr != nil {
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal config: %w", err)
		} else {
			zap.L().Warn("unknown fields in config file, ignoring", zap.Error(strictErr))
		}
	}

	return &config, nil
}

func (f *JsonFileStore[T]) Update(config *v1.Config) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	data, err := protojson.MarshalOptions{
		Indent:    "  ",
		Multiline: true,
	}.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	err = os.MkdirAll(filepath.Dir(f.Path), 0755)
	if err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	// backup the old config file
	if err := f.makeBackup(); err != nil {
		return fmt.Errorf("backup config file: %w", err)
	}

	err = atomic.WriteFile(f.Path, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	// only the user running backrest should be able to read the config.
	if err := os.Chmod(f.Path, 0600); err != nil {
		return fmt.Errorf("chmod(0600) config file: %w", err)
	}

	return nil
}

func (f *JsonFileStore[T]) makeBackup() error {
	curConfig, err := os.ReadFile(f.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	// backup the current config file
	backupName := fmt.Sprintf("%s.bak.%s", f.Path, time.Now().Format("2006-01-02-15-04-05"))
	if err := atomic.WriteFile(backupName, bytes.NewBuffer(curConfig)); err != nil {
		return err
	}
	if err := os.Chmod(backupName, 0600); err != nil {
		return err
	}

	// only keep the last 10 versions
	files, err := filepath.Glob(f.Path + ".bak.*")
	if err != nil {
		return err
	}
	if len(files) > configKeepVersions {
		for _, file := range files[:len(files)-configKeepVersions] {
			if err := os.Remove(file); err != nil {
				return err
			}
		}
	}

	return nil
}
