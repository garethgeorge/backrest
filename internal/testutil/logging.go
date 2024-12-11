package testutil

import (
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type testLogger struct {
	t *testing.T
}

func (l *testLogger) Write(p []byte) (n int, err error) {
	l.t.Log("global log: " + strings.Trim(string(p), "\n"))
	return len(p), nil
}

func InstallZapLogger(t *testing.T) {
	t.Helper()
	logger := zap.New(zapcore.NewCore(
		zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
		zapcore.AddSync(&testLogger{t: t}),
		zapcore.DebugLevel,
	))
	zap.ReplaceGlobals(logger)
}
