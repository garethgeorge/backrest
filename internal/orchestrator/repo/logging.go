package repo

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/garethgeorge/backrest/internal/orchestrator/logging"
	"github.com/garethgeorge/backrest/pkg/restic"
)

// pipeResticLogsToWriter sets the restic logger to write to the provided writer.
// returns a new context with the logger set and a function to flush the logs.
func forwardResticLogs(ctx context.Context) (context.Context, func()) {
	writer := logging.WriterFromContext(ctx)
	if writer == nil {
		return ctx, func() {}
	}
	limit := &limitWriter{w: writer, n: 64 * 1024}
	capture := &linePrefixer{w: limit, prefix: []byte("[restic] ")}
	return restic.ContextWithLogger(ctx, capture), func() {
		if limit.d > 0 {
			fmt.Fprintf(writer, "Output truncated, %d bytes dropped\n", limit.d)
		}
		capture.Close()
	}
}

// limitWriter is a writer that limits the number of bytes written to it.
type limitWriter struct {
	w io.Writer
	n int
	d int
}

func (l *limitWriter) Write(p []byte) (rnw int, err error) {
	rnw = len(p)
	if l.n <= 0 {
		l.d += len(p)
		return 0, nil
	}
	if len(p) > l.n {
		l.d += len(p) - l.n
		p = p[:l.n]
	}
	_, err = l.w.Write(p)
	l.n -= len(p)
	return
}

type linePrefixer struct {
	w      io.Writer
	buf    []byte
	prefix []byte
}

func (l *linePrefixer) Write(p []byte) (n int, err error) {
	n = len(p)
	l.buf = append(l.buf, p...)

	bufOrig := l.buf
	wroteLines := false
	for {
		i := bytes.IndexByte(l.buf, '\n')
		if i < 0 {
			break
		}
		wroteLines = true
		if _, err := l.w.Write(l.prefix); err != nil {
			return 0, err
		}
		if _, err := l.w.Write(l.buf[:i+1]); err != nil {
			return 0, err
		}
		l.buf = l.buf[i+1:]
	}
	if wroteLines {
		l.buf = append(bufOrig[:0], l.buf...)
	}
	return
}

func (l *linePrefixer) Close() error {
	if len(l.buf) > 0 {
		if _, err := l.w.Write(l.prefix); err != nil {
			return err
		}
		if _, err := l.w.Write(l.buf); err != nil {
			return err
		}
	}
	return nil
}
