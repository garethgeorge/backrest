package ioutil

import (
	"fmt"
	"io"
)

// HeadWriter keeps the first 'Limit' bytes in memory.
type HeadWriter struct {
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
	return w.Buf
}

// tailWriter keeps the last 'Limit' bytes in memory.
type TailWriter struct {
	Buf   []byte
	Limit int
}

var _ io.Writer = &TailWriter{}

func (w *TailWriter) Write(p []byte) (n int, err error) {
	w.Buf = append(w.Buf, p...)
	if len(w.Buf) > w.Limit {
		w.Buf = w.Buf[len(w.Buf)-w.Limit:]
	}
	return len(p), nil
}

func (w *TailWriter) Bytes() []byte {
	return w.Buf
}

type OutputCapturer struct {
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
	w.HeadWriter.Write(p)
	w.TailWriter.Write(p)
	w.totalBytes += len(p)
	return len(p), nil
}

func (w *OutputCapturer) String() string {
	head := w.HeadWriter.Bytes()
	tail := w.TailWriter.Bytes()
	if w.totalBytes <= w.Limit {
		return string(head)
	}

	head = head[:w.Limit/2]
	tail = tail[len(tail)-w.Limit/2:]

	return fmt.Sprintf("%s...[%v bytes dropped]...%s", string(head), w.totalBytes-len(head)-len(tail), string(tail))
}
