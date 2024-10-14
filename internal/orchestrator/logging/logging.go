package logging

import (
	"context"
	"io"

	"github.com/garethgeorge/backrest/internal/ioutil"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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

// Logger returns a logger from the context, or the global logger if none is found.
// this is somewhat expensive, it should be called once per task.
func Logger(ctx context.Context, prefix string) *zap.Logger {
	writer := WriterFromContext(ctx)
	if writer == nil {
		return zap.L()
	}
	p := zap.NewProductionEncoderConfig()
	p.EncodeTime = zapcore.TimeEncoderOfLayout("15:04:05.000Z")
	fe := zapcore.NewConsoleEncoder(p)
	l := zap.New(zapcore.NewTee(
		zap.L().Core(),
		zapcore.NewCore(fe, zapcore.AddSync(&ioutil.LinePrefixer{W: writer, Prefix: []byte(prefix)}), zapcore.DebugLevel),
	))
	return l
}
