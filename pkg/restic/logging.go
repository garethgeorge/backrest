package restic

import (
	"context"
	"io"
	"os/exec"
)

var loggerKey = struct{}{}

func ContextWithLogger(ctx context.Context, logger io.Writer) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

func LoggerFromContext(ctx context.Context) io.Writer {
	writer, _ := ctx.Value(loggerKey).(io.Writer)
	return writer
}

func addLoggingToCommand(ctx context.Context, cmd *exec.Cmd) {
	logger := LoggerFromContext(ctx)
	if logger == nil {
		return
	}
	if cmd.Stdout != nil {
		cmd.Stdout = io.MultiWriter(cmd.Stdout, logger)
	}
	if cmd.Stderr != nil {
		cmd.Stderr = io.MultiWriter(cmd.Stderr, logger)
	}
}
