package restic

import "io"

// limitWriter silently stops writing after 'limit' bytes.
type limitWriter struct {
	written int64
	limit   int64
	w       io.Writer
}

var _ io.Writer = &limitWriter{}

func (w *limitWriter) Write(p []byte) (n int, err error) {
	r := len(p)
	if w.written >= w.limit {
		return r, nil
	}

	if w.written+int64(len(p)) > w.limit {
		p = p[:w.limit-w.written]
	}

	n, err = w.w.Write(p)
	w.written += int64(n)
	return r, err
}

func newLimitWriter(w io.Writer, limit int64) io.Writer {
	return &limitWriter{
		w:     w,
		limit: limit,
	}
}
