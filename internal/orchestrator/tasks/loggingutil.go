package tasks

import (
	"context"

	"github.com/garethgeorge/backrest/internal/orchestrator/logging"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LoggerFromContext returns a logger from the context, or the global logger if none is found.
// this is somewhat expensive, it should be called once per task.
func Logger(ctx context.Context, task Task) *zap.Logger {
	writer := logging.WriterFromContext(ctx)
	if writer == nil {
		return zap.L()
	}
	p := zap.NewProductionEncoderConfig()
	p.EncodeTime = zapcore.ISO8601TimeEncoder
	fe := zapcore.NewJSONEncoder(p)
	l := zap.New(zapcore.NewTee(
		zap.L().Core(),
		zapcore.NewCore(fe, zapcore.AddSync(writer), zapcore.DebugLevel),
	))
	return l
}
