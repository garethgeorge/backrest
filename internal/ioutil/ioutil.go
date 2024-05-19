package ioutil

import (
	"bytes"
	"fmt"
	"io"
	"slices"
	"sync"
)

type Capturer interface {
	Bytes() []byte
}

// HeadWriter keeps the first 'Limit' bytes in memory.
type HeadWriter struct {
	mu    sync.Mutex
	Buf   []byte
	Limit int
}

var _ io.Writer = &HeadWriter{}

func (w *HeadWriter) Write(p []byte) (n int, err error) {
	if len(w.Buf) >= w.Limit {
		return len(p), nil
	}
	w.Buf = append(w.Buf, p...)
	if len(w.Buf) > w.Limit {
		w.Buf = w.Buf[:w.Limit]
	}
	return len(p), nil
}

func (w *HeadWriter) Bytes() []byte {
	w.mu.Lock()
	defer w.mu.Unlock()
	return slices.Clone(w.Buf)
}

// tailWriter keeps the last 'Limit' bytes in memory.
type TailWriter struct {
	mu    sync.Mutex
	Buf   []byte
	Limit int
}

var _ io.Writer = &TailWriter{}

func (w *TailWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.Buf = append(w.Buf, p...)
	if len(w.Buf) > w.Limit {
		w.Buf = w.Buf[len(w.Buf)-w.Limit:]
	}
	return len(p), nil
}

func (w *TailWriter) Bytes() []byte {
	w.mu.Lock()
	defer w.mu.Unlock()
	return slices.Clone(w.Buf)
}

type OutputCapturer struct {
	mu sync.Mutex
	HeadWriter
	TailWriter
	Limit      int
	totalBytes int
}

var _ io.Writer = &OutputCapturer{}

func NewOutputCapturer(limit int) *OutputCapturer {
	return &OutputCapturer{
		HeadWriter: HeadWriter{Limit: limit},
		TailWriter: TailWriter{Limit: limit},
		Limit:      limit,
	}
}

func (w *OutputCapturer) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.HeadWriter.Write(p)
	w.TailWriter.Write(p)
	w.totalBytes += len(p)
	return len(p), nil
}

func (w *OutputCapturer) Bytes() []byte {
	w.mu.Lock()
	defer w.mu.Unlock()
	head := w.HeadWriter.Bytes()
	tail := w.TailWriter.Bytes()
	if w.totalBytes <= w.Limit {
		return head
	}

	head = head[:w.Limit/2]
	tail = tail[len(tail)-w.Limit/2:]

	buf := bytes.NewBuffer(make([]byte, 0, len(head)+len(tail)+100))

	buf.Write(head)
	buf.WriteString(fmt.Sprintf("...[%v bytes dropped]...", w.totalBytes-len(head)-len(tail)))
	buf.Write(tail)

	return buf.Bytes()
}

type SynchronizedWriter struct {
	Mu sync.Mutex
	W  io.Writer
}

var _ io.Writer = &SynchronizedWriter{}

func (w *SynchronizedWriter) Write(p []byte) (n int, err error) {
	w.Mu.Lock()
	defer w.Mu.Unlock()
	return w.W.Write(p)
}
