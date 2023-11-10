package eventlog

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
)

// SimpleLogFile is a simple log file implementation that appends to a file.
type SimpleLogFile struct {
	file   string   // path to the log file
	handle *os.File // file handle
	mu     sync.Mutex
}

func NewSimpleLogFile(file string) *SimpleLogFile {
	return &SimpleLogFile{
		file: file,
	}
}

var _ LogFile = &SimpleLogFile{}

func (s *SimpleLogFile) open() error {
	if s.handle != nil {
		return nil
	}

	err := os.MkdirAll(filepath.Dir(s.file), 0755)
	if err != nil {
		return fmt.Errorf("failed to create parent dirs of log file %s: %w", s.file, err)
	}

	f, err := os.OpenFile(s.file, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)

	if err != nil {
		return fmt.Errorf("failed to open log file %s: %w", s.file, err)
	}

	s.handle = f

	return nil
}

func (s *SimpleLogFile) Log(event interface{}) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	err = s.open()
	if err != nil {
		return err
	}

	s.handle.Write(data)
	s.handle.Write([]byte("\n"))

	return nil
}

func (s *SimpleLogFile) Size() (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := s.open()
	if err != nil {
		return 0, err
	}

	stat, err := s.handle.Stat()
	if err != nil {
		return 0, fmt.Errorf("failed to stat log file %s: %w", s.file, err)
	}

	return int(stat.Size()), nil
}

func (s *SimpleLogFile) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.handle == nil {
		return nil
	}

	err := s.handle.Close()
	if err != nil {
		return fmt.Errorf("failed to close log file %s: %w", s.file, err)
	}
	s.handle = nil

	return nil
}

// Iterator returns a function that can be called to iterate over the log file. 
func (s *SimpleLogFile) Iterator() (LogIterator, error) {
	s.mu.Lock()
	f, err := os.OpenFile(s.file, os.O_RDONLY, 0644)
	if err != nil {
		s.mu.Unlock()
		if errors.Is(err, os.ErrNotExist) {
			return &funcLogIterator{
				nextFunc: func() interface{} {
					return nil
				},
				closeFunc: func() error {
					return nil
				},
			}, nil
		}
		return nil, fmt.Errorf("failed to open log file %s: %w", s.file, err)
	}

	stat, err := f.Stat()
	if err != nil {
		s.mu.Unlock()
		return nil, fmt.Errorf("failed to stat log file %s: %w", s.file, err)
	}
	size := int(stat.Size())
	s.mu.Unlock()

	reader := io.NewSectionReader(f, 0, int64(size))
	scanner := bufio.NewScanner(reader)

	nextFunc := func() interface{} {
		if !scanner.Scan() {
			return nil
		}

		var event interface{}
		err := json.Unmarshal(scanner.Bytes(), &event)
		if err != nil {
			log.Default().Printf("failed to unmarshal event: %v", err)
			return nil
		}

		return event
	}

	closeFunc := func() error {
		return f.Close()
	}

	return &funcLogIterator{
		nextFunc: nextFunc,
		closeFunc: closeFunc,
	}, nil
}
