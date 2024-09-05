package logwriter

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path"
	"slices"
	"sync"
)

var ErrAlreadyExists = errors.New("already exists")

type LiveLog struct {
	mu      sync.Mutex
	dir     string
	writers map[string]*LiveLogWriter
}

func NewLiveLogger(dir string) (*LiveLog, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return &LiveLog{dir: dir, writers: make(map[string]*LiveLogWriter)}, nil
}

func (t *LiveLog) ListIDs() []string {
	t.mu.Lock()
	defer t.mu.Unlock()

	files, err := os.ReadDir(t.dir)
	if err != nil {
		return nil
	}

	ids := make([]string, 0, len(files))
	for _, f := range files {
		if !f.IsDir() {
			ids = append(ids, f.Name())
		}
	}
	return ids
}

func (t *LiveLog) NewWriter(id string) (*LiveLogWriter, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, ok := t.writers[id]; ok {
		return nil, ErrAlreadyExists
	}
	fh, err := os.Create(path.Join(t.dir, id))
	if err != nil {
		return nil, err
	}
	w := &LiveLogWriter{
		fh:   fh,
		id:   id,
		ll:   t,
		path: path.Join(t.dir, id),
	}
	t.writers[id] = w
	return w, nil
}

func (t *LiveLog) Unsubscribe(id string, ch chan []byte) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if w, ok := t.writers[id]; ok {
		w.mu.Lock()
		defer w.mu.Unlock()

		idx := slices.IndexFunc(w.subscribers, func(c chan []byte) bool {
			return c == ch
		})
		if idx >= 0 {
			close(ch)
			w.subscribers = append(w.subscribers[:idx], w.subscribers[idx+1:]...)
		}
	}
}

func (t *LiveLog) Subscribe(id string) (chan []byte, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if w, ok := t.writers[id]; ok {
		// If there is a writer, block writes until we are done opening the file
		w.mu.Lock()
		defer w.mu.Unlock()
	}

	fh, err := os.Open(path.Join(t.dir, id))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrFileNotFound
		}
		return nil, err
	}

	ch := make(chan []byte, 1)
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := fh.Read(buf)
			if err == io.EOF {
				break
			} else if err != nil {
				return
			}
			ch <- bytes.Clone(buf[:n])
		}

		// Lock the writer to prevent writes while we switch subscription modes
		t.mu.Lock()
		if w, ok := t.writers[id]; ok {
			w.mu.Lock()
			defer w.mu.Unlock()
		}
		t.mu.Unlock()

		// Read anything written while we were acquiring the lock
		for {
			n, err := fh.Read(buf)
			if err == io.EOF {
				break
			}
			if err != nil {
				close(ch)
				fh.Close()
				return
			}
			ch <- bytes.Clone(buf[:n])
		}
		fh.Close()

		// Install subscription in the writer OR close the channel if the writer is gone
		t.mu.Lock()
		if w, ok := t.writers[id]; ok {
			w.subscribers = append(w.subscribers, ch)
		} else {
			close(ch)
		}
		t.mu.Unlock()
	}()

	return ch, nil
}

func (t *LiveLog) Remove(id string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.writers, id)
	return os.Remove(path.Join(t.dir, id))
}

func (t *LiveLog) IsAlive(id string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	_, ok := t.writers[id]
	return ok
}

type LiveLogWriter struct {
	mu          sync.Mutex
	ll          *LiveLog
	fh          *os.File
	id          string
	path        string
	subscribers []chan []byte
}

func (t *LiveLogWriter) Write(data []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	n, err := t.fh.Write(data)
	if err != nil {
		return 0, err
	}
	if n != len(data) {
		return n, errors.New("short write")
	}
	for _, ch := range t.subscribers {
		ch <- bytes.Clone(data)
	}
	return n, nil
}

func (t *LiveLogWriter) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.ll.mu.Lock()
	defer t.ll.mu.Unlock()
	delete(t.ll.writers, t.id)

	for _, ch := range t.subscribers {
		close(ch)
	}
	t.subscribers = nil

	return t.fh.Close()
}
