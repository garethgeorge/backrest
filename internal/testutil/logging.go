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
	l.t.Log(strings.Trim(string(p), "\n"))
	return len(p), nil
}

// InstallZapLogger sets up a global zap logger for testing purposes.
func InstallZapLogger(t *testing.T) {
	t.Helper()
	zap.ReplaceGlobals(NewTestLogger(t))
}

// NewTestLogger creates a zap logger for testing that outputs to the test log.
func NewTestLogger(t *testing.T) *zap.Logger {
	t.Helper()
	return zap.New(zapcore.NewCore(
		zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
		zapcore.AddSync(&testLogger{t: t}),
		zapcore.DebugLevel,
	))
}
