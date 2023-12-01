package orchestrator

import (
	"bytes"
	"sync"
)

func pruneHelper() {
	// TODO: This is a stub.

}

// synchronizedBuffer is used for collecting prune command's output
type synchronizedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (w *synchronizedBuffer) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.buf.Write(p)
}

func (w *synchronizedBuffer) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.buf.String()
}
