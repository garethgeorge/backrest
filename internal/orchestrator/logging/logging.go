package logging

import (
	"context"
	"io"
)

type contextKey int

const (
	contextKeyLogWriter contextKey = iota
)

func WriterFromContext(ctx context.Context) io.Writer {
	writer, ok := ctx.Value(contextKeyLogWriter).(io.Writer)
	if !ok {
		return nil
	}
	return writer
}

func ContextWithWriter(ctx context.Context, logger io.Writer) context.Context {
	return context.WithValue(ctx, contextKeyLogWriter, logger)
}
