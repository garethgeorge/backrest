package orchestrator

import (
	"bytes"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/ioutil"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// taskRunnerImpl is an implementation of TaskRunner for the default orchestrator.
type taskRunnerImpl struct {
	orchestrator *Orchestrator
	st           ScheduledTask
	logs         bytes.Buffer
	logsWriter   ioutil.LockedWriter
}

var _ TaskRunner = &taskRunnerImpl{}

func newTaskRunnerImpl(orchestrator *Orchestrator, task ScheduledTask) *taskRunnerImpl {
	impl := &taskRunnerImpl{
		orchestrator: orchestrator,
		st:           task,
	}
	impl.logsWriter = ioutil.LockedWriter{W: &impl.logs}
	return impl
}

func (t *taskRunnerImpl) CreateOperation(op *v1.Operation) error {
	return t.orchestrator.OpLog.Add(op)
}

func (t *taskRunnerImpl) UpdateOperation(op *v1.Operation) error {
	return t.orchestrator.OpLog.Update(op)
}

// Logger returns a logger for the run of the task.
// It will log to the default logger and a file for this task.
func (t *taskRunnerImpl) Logger() *zap.Logger {
	p := zap.NewProductionEncoderConfig()
	p.EncodeTime = zapcore.ISO8601TimeEncoder
	fe := zapcore.NewJSONEncoder(p)
	l := zap.New(zapcore.NewTee(
		zap.L().Core(),
		zapcore.NewCore(fe, zapcore.AddSync(&t.logsWriter), zapcore.DebugLevel),
	))
	return l.Named(t.st.Task.Name())
}

func (t *taskRunnerImpl) AppendRawLog(data []byte) error {
	_, err := t.logsWriter.Write(data)
	return err
}

func (t *taskRunnerImpl) Orchestrator() *Orchestrator {
	return t.orchestrator
}
