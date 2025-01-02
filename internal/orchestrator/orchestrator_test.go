package orchestrator

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/config"
	"github.com/garethgeorge/backrest/internal/cryptoutil"
	"github.com/garethgeorge/backrest/internal/orchestrator/tasks"
)

type testTask struct {
	tasks.BaseTask
	onRun  func() error
	onNext func(curTime time.Time) *time.Time
}

var _ tasks.Task = &testTask{}

func newTestTask(onRun func() error, onNext func(curTime time.Time) *time.Time) tasks.Task {
	return &testTask{
		BaseTask: tasks.BaseTask{
			TaskName:   "test task",
			TaskRepo:   &v1.Repo{Id: "repo", Guid: cryptoutil.MustRandomID(cryptoutil.DefaultIDBits)},
			TaskPlanID: "plan",
		},
		onRun:  onRun,
		onNext: onNext,
	}
}
func (t *testTask) Next(curTime time.Time, runner tasks.TaskRunner) (tasks.ScheduledTask, error) {
	at := t.onNext(curTime)
	if at == nil {
		return tasks.NeverScheduledTask, nil
	}
	return tasks.ScheduledTask{
		Task:  t,
		RunAt: *at,
	}, nil
}

func (t *testTask) Run(ctx context.Context, st tasks.ScheduledTask, runner tasks.TaskRunner) error {
	return t.onRun()
}

func TestTaskScheduling(t *testing.T) {
	t.Parallel()

	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	orch, err := NewOrchestrator("", config.NewDefaultConfig(), nil, nil)
	if err != nil {
		t.Fatalf("failed to create orchestrator: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	task := newTestTask(
		func() error {
			wg.Done()
			cancel()
			return nil
		},
		func(t time.Time) *time.Time {
			t = t.Add(10 * time.Millisecond)
			return &t
		},
	)

	wg.Add(1)
	go func() {
		defer wg.Done()
		orch.Run(ctx)
	}()

	// Act
	orch.ScheduleTask(task, tasks.TaskPriorityDefault)

	// Assert passes if all tasks run and the orchestrator exists when cancelled.
	wg.Wait()
}

func TestTaskRescheduling(t *testing.T) {
	t.Parallel()

	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	orch, err := NewOrchestrator("", config.NewDefaultConfig(), nil, nil)
	if err != nil {
		t.Fatalf("failed to create orchestrator: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		orch.Run(ctx)
	}()

	// Act
	count := 0
	ranTimes := 0

	orch.ScheduleTask(newTestTask(
		func() error {
			ranTimes += 1
			if ranTimes == 10 {
				cancel()
			}
			return nil
		},
		func(t time.Time) *time.Time {
			if count < 10 {
				count += 1
				return &t
			}
			return nil
		},
	), tasks.TaskPriorityDefault)

	wg.Wait()

	if count != 10 {
		t.Errorf("expected 10 Next calls, got %d", count)
	}

	if ranTimes != 10 {
		t.Errorf("expected 10 Run calls, got %d", ranTimes)
	}
}

func TestTaskRetry(t *testing.T) {
	t.Parallel()

	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	orch, err := NewOrchestrator("", config.NewDefaultConfig(), nil, nil)
	if err != nil {
		t.Fatalf("failed to create orchestrator: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		orch.Run(ctx)
	}()

	// Act
	count := 0
	ranTimes := 0

	orch.ScheduleTask(newTestTask(
		func() error {
			ranTimes += 1
			if ranTimes == 10 {
				cancel()
			}
			return &tasks.TaskRetryError{
				Err:     errors.New("retry please"),
				Backoff: func(attempt int) time.Duration { return 0 },
			}
		},
		func(t time.Time) *time.Time {
			count += 1
			return &t
		},
	), tasks.TaskPriorityDefault)

	wg.Wait()

	if count != 1 {
		t.Errorf("expected 1 Next calls because this test covers retries, got %d", count)
	}

	if ranTimes != 10 {
		t.Errorf("expected 10 Run calls, got %d", ranTimes)
	}
}

func TestGracefulShutdown(t *testing.T) {
	t.Parallel()

	// Arrange
	orch, err := NewOrchestrator("", config.NewDefaultConfig(), nil, nil)
	if err != nil {
		t.Fatalf("failed to create orchestrator: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	// Act
	orch.Run(ctx)
}

func TestSchedulerWait(t *testing.T) {
	t.Parallel()

	// Arrange
	orch, err := NewOrchestrator("", config.NewDefaultConfig(), nil, nil)
	if err != nil {
		t.Fatalf("failed to create orchestrator: %v", err)
	}
	orch.taskQueue.Reset()

	ran := make(chan struct{})
	didRun := false
	orch.ScheduleTask(newTestTask(
		func() error {
			close(ran)
			return nil
		},
		func(t time.Time) *time.Time {
			if didRun {
				return nil
			}
			t = t.Add(100 * time.Millisecond)
			didRun = true
			return &t
		},
	), tasks.TaskPriorityDefault)

	// Act
	go orch.Run(context.Background())

	// Assert
	select {
	case <-time.NewTimer(20 * time.Millisecond).C:
	case <-ran:
		t.Errorf("expected task to not run yet")
	}

	// Schedule another task just to trigger a queue refresh
	orch.ScheduleTask(&testTask{
		onNext: func(t time.Time) *time.Time {
			t = t.Add(1000 * time.Second)
			return &t
		},
		onRun: func() error {
			t.Fatalf("should never run")
			return nil
		},
	}, tasks.TaskPriorityDefault)

	select {
	case <-time.NewTimer(200 * time.Millisecond).C:
		t.Errorf("expected task to run")
	case <-ran:
	}
}
