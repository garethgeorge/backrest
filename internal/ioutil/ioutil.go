package ioutil

import (
	"bytes"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
)

// LimitWriter is a writer that limits the number of bytes written to it.
type LimitWriter struct {
	W io.Writer
	N int // bytes remaining that can be written
	D int // bytes dropped so far
}

func (l *LimitWriter) Write(p []byte) (rnw int, err error) {
	rnw = len(p)
	if l.N <= 0 {
		l.D += len(p)
		return 0, nil
	}
	if len(p) > l.N {
		l.D += len(p) - l.N
		p = p[:l.N]
	}
	_, err = l.W.Write(p)
	l.N -= len(p)
	return
}

// LinePrefixer is a writer that prefixes each line written to it with a prefix.
type LinePrefixer struct {
	W      io.Writer
	buf    []byte
	Prefix []byte
}

func (l *LinePrefixer) Write(p []byte) (n int, err error) {
	n = len(p)
	l.buf = append(l.buf, p...)
	if !bytes.Contains(p, []byte{'\n'}) { // no newlines in p, short-circuit out
		return
	}
	bufOrig := l.buf
	for {
		i := bytes.IndexByte(l.buf, '\n')
		if i < 0 {
			break
		}
		if _, err := l.W.Write(l.Prefix); err != nil {
			return 0, err
		}
		if _, err := l.W.Write(l.buf[:i+1]); err != nil {
			return 0, err
		}
		l.buf = l.buf[i+1:]
	}
	l.buf = append(bufOrig[:0], l.buf...)
	return
}

func (l *LinePrefixer) Close() error {
	if len(l.buf) > 0 {
		if _, err := l.W.Write(l.Prefix); err != nil {
			return err
		}
		if _, err := l.W.Write(l.buf); err != nil {
			return err
		}
	}
	return nil
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

type SizeTrackingWriter struct {
	size atomic.Uint64
	io.Writer
}

func (w *SizeTrackingWriter) Write(p []byte) (n int, err error) {
	n, err = w.Writer.Write(p)
	w.size.Add(uint64(n))
	return
}

// Size returns the number of bytes written to the writer.
// The value is fundamentally racy only consistent if synchronized with the writer or closed.
func (w *SizeTrackingWriter) Size() uint64 {
	return w.size.Load()
}

type SizeLimitedWriter struct {
	SizeTrackingWriter
	Limit uint64
}

var _ io.Writer = &SizeLimitedWriter{}

func (w *SizeLimitedWriter) Write(p []byte) (n int, err error) {
	size := w.Size()
	if size+uint64(len(p)) > w.Limit {
		p = p[:w.Limit-size]
		err = fmt.Errorf("size limit exceeded: %d bytes written, limit is %d bytes", size, w.Limit)
	}

	var e error
	n, e = w.Writer.Write(p)
	if e != nil {
		err = e
	}
	return
}
