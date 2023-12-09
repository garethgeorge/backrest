package restic

import (
	"fmt"
	"io"
)

// headWriter keeps the first 'limit' bytes in memory.
type headWriter struct {
	buf   []byte
	limit int
}

var _ io.Writer = &headWriter{}

func (w *headWriter) Write(p []byte) (n int, err error) {
	if len(w.buf) >= w.limit {
		return len(p), nil
	}
	w.buf = append(w.buf, p...)
	if len(w.buf) > w.limit {
		w.buf = w.buf[:w.limit]
	}
	return len(p), nil
}

func (w *headWriter) Bytes() []byte {
	return w.buf
}

// tailWriter keeps the last 'limit' bytes in memory.
type tailWriter struct {
	buf   []byte
	limit int
}

var _ io.Writer = &tailWriter{}

func (w *tailWriter) Write(p []byte) (n int, err error) {
	w.buf = append(w.buf, p...)
	if len(w.buf) > w.limit {
		w.buf = w.buf[len(w.buf)-w.limit:]
	}
	return len(p), nil
}

func (w *tailWriter) Bytes() []byte {
	return w.buf
}

type outputCapturer struct {
	headWriter
	tailWriter
	limit      int
	totalBytes int
}

var _ io.Writer = &outputCapturer{}

func newOutputCapturer(limit int) *outputCapturer {
	return &outputCapturer{
		headWriter: headWriter{limit: limit},
		tailWriter: tailWriter{limit: limit},
		limit:      limit,
	}
}

func (w *outputCapturer) Write(p []byte) (n int, err error) {
	w.headWriter.Write(p)
	w.tailWriter.Write(p)
	w.totalBytes += len(p)
	return len(p), nil
}

func (w *outputCapturer) String() string {
	head := w.headWriter.Bytes()
	tail := w.tailWriter.Bytes()
	if w.totalBytes <= w.limit {
		return string(head)
	}

	head = head[:w.limit/2]
	tail = tail[len(tail)-w.limit/2:]

	return fmt.Sprintf("%s...[%v bytes dropped]...%s", string(head), w.totalBytes-len(head)-len(tail), string(tail))
}
