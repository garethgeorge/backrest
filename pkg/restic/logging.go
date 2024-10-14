package restic

import (
	"context"
	"io"
)

var loggerKey = struct{}{}

func ContextWithLogger(ctx context.Context, logger io.Writer) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

func LoggerFromContext(ctx context.Context) io.Writer {
	writer, _ := ctx.Value(loggerKey).(io.Writer)
	return writer
}
