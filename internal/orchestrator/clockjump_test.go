package orchestrator

import (
	"errors"
	"strings"
	"testing"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/config"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/oplog/memstore"
	"github.com/garethgeorge/backrest/internal/orchestrator/tasks"
	"google.golang.org/protobuf/proto"
)

type clockJumpBackupFixture struct {
	orch    *Orchestrator
	cfg     *v1.Config
	log     *oplog.OpLog
	running stContainer
	now     time.Time
}

func newClockJumpBackupFixture(t *testing.T) clockJumpBackupFixture {
	t.Helper()
	now := time.Now().Truncate(time.Millisecond)
	repo := &v1.Repo{Id: "repo1", Guid: strings.Repeat("a", 64), Uri: "/tmp/repo"}
	plan := &v1.Plan{
		Id:    "plan1",
		Repo:  repo.Id,
		Paths: []string{"/tmp/backup"},
		Schedule: &v1.Schedule{
			Schedule: &v1.Schedule_MaxFrequencyHours{MaxFrequencyHours: 12},
			Clock:    v1.Schedule_CLOCK_LAST_RUN_TIME,
		},
	}
	cfg := config.NewDefaultConfig()
	cfg.Instance = "instance1"
	cfg.Repos = []*v1.Repo{repo}
	cfg.Plans = []*v1.Plan{plan}

	log, err := oplog.NewOpLog(memstore.NewMemStore())
	if err != nil {
		t.Fatal(err)
	}
	previousEnd := now.Add(-13 * time.Hour)
	if err := log.Add(&v1.Operation{
		InstanceId:      cfg.Instance,
		RepoId:          repo.Id,
		RepoGuid:        repo.Guid,
		PlanId:          plan.Id,
		Status:          v1.OperationStatus_STATUS_SUCCESS,
		UnixTimeStartMs: previousEnd.Add(-time.Minute).UnixMilli(),
		UnixTimeEndMs:   previousEnd.UnixMilli(),
		Op:              &v1.Operation_OperationBackup{OperationBackup: &v1.OperationBackup{}},
	}); err != nil {
		t.Fatal(err)
	}

	mgr := &config.ConfigManager{Store: &config.MemoryStore{Config: cfg}}
	orch, err := NewOrchestrator("", mgr, log, nil)
	if err != nil {
		t.Fatal(err)
	}
	var running stContainer
	for _, queued := range queuedBackups(orch, plan.Id) {
		running = queued
		break
	}
	if running.Task == nil {
		t.Fatal("overdue backup was not scheduled")
	}
	if !running.RunAt.Before(now) {
		t.Fatalf("backup was not overdue: %v >= %v", running.RunAt, now)
	}
	orch.taskQueue.Remove(running) // The task has left the queue and is running.
	return clockJumpBackupFixture{orch: orch, cfg: cfg, log: log, running: running, now: now}
}

func TestClockJumpDoesNotDuplicateRunningBackup(t *testing.T) {
	for _, status := range []v1.OperationStatus{
		v1.OperationStatus_STATUS_PENDING,    // Dequeued, before RunTask marks it in progress.
		v1.OperationStatus_STATUS_INPROGRESS, // Backup already running.
	} {
		t.Run(status.String(), func(t *testing.T) {
			f := newClockJumpBackupFixture(t)
			f.running.Op.Status = status
			f.running.Op.UnixTimeStartMs = f.now.UnixMilli()
			if err := f.log.Update(f.running.Op); err != nil {
				t.Fatal(err)
			}
			if err := f.orch.ScheduleDefaultTasks(f.cfg); err != nil {
				t.Fatal(err)
			}
			if backups := queuedBackups(f.orch, "plan1"); len(backups) != 0 {
				t.Fatalf("queue reset duplicated an active backup: %d queued", len(backups))
			}

			finished := f.now.Add(5 * time.Minute)
			f.running.Op.Status = v1.OperationStatus_STATUS_SUCCESS
			f.running.Op.UnixTimeEndMs = finished.UnixMilli()
			if err := f.log.Update(f.running.Op); err != nil {
				t.Fatal(err)
			}
			f.orch.handleTaskCompletion(&f.running, nil, nil)
			backups := queuedBackups(f.orch, "plan1")
			if len(backups) != 1 {
				t.Fatalf("expected one next backup after completion, got %d", len(backups))
			}
			want := finished.Add(12 * time.Hour)
			if !backups[0].RunAt.Equal(want) {
				t.Fatalf("next run = %v, want %v", backups[0].RunAt, want)
			}
		})
	}
}

func TestQueueResetAfterBackupFinishesDoesNotDuplicateNextRun(t *testing.T) {
	f := newClockJumpBackupFixture(t)
	finished := f.now.Add(5 * time.Minute)
	f.running.Op.Status = v1.OperationStatus_STATUS_SUCCESS
	f.running.Op.UnixTimeEndMs = finished.UnixMilli()
	if err := f.log.Update(f.running.Op); err != nil {
		t.Fatal(err)
	}
	if err := f.orch.ScheduleDefaultTasks(f.cfg); err != nil {
		t.Fatal(err)
	}
	f.orch.handleTaskCompletion(&f.running, nil, nil)
	backups := queuedBackups(f.orch, "plan1")
	if len(backups) != 1 {
		t.Fatalf("expected one next backup, got %d", len(backups))
	}
	if want := finished.Add(12 * time.Hour); !backups[0].RunAt.Equal(want) {
		t.Fatalf("next run = %v, want %v", backups[0].RunAt, want)
	}
}

func TestQueueResetDoesNotRestoreRemovedBackupPlan(t *testing.T) {
	f := newClockJumpBackupFixture(t)
	f.running.Op.Status = v1.OperationStatus_STATUS_INPROGRESS
	if err := f.log.Update(f.running.Op); err != nil {
		t.Fatal(err)
	}
	updated := proto.Clone(f.cfg).(*v1.Config)
	updated.Plans = nil
	if err := f.orch.applyConfig(updated); err != nil {
		t.Fatal(err)
	}
	f.running.Op.Status = v1.OperationStatus_STATUS_SUCCESS
	f.running.Op.UnixTimeEndMs = f.now.Add(5 * time.Minute).UnixMilli()
	if err := f.log.Update(f.running.Op); err != nil {
		t.Fatal(err)
	}
	f.orch.handleTaskCompletion(&f.running, nil, nil)
	if backups := queuedBackups(f.orch, "plan1"); len(backups) != 0 {
		t.Fatalf("removed plan was rescheduled: %d backups", len(backups))
	}
}

func TestQueueResetPreservesBackupRetry(t *testing.T) {
	f := newClockJumpBackupFixture(t)
	original := proto.Clone(f.running.Op).(*v1.Operation)
	f.running.Op.Status = v1.OperationStatus_STATUS_INPROGRESS
	if err := f.log.Update(f.running.Op); err != nil {
		t.Fatal(err)
	}
	if err := f.orch.ScheduleDefaultTasks(f.cfg); err != nil {
		t.Fatal(err)
	}
	retryErr := &tasks.TaskRetryError{
		Err:     errors.New("temporary backup failure"),
		Backoff: func(int) time.Duration { return time.Hour },
	}
	if !f.orch.handleTaskCompletion(&f.running, retryErr, original) {
		t.Fatal("expected the active backup to be retried")
	}
	retries := queuedBackups(f.orch, "plan1")
	if len(retries) != 1 || retries[0].Op.GetId() != f.running.Op.GetId() {
		t.Fatalf("expected one retry of the same operation, got %d", len(retries))
	}
	if delay := retries[0].RunAt.Sub(f.now); delay < 59*time.Minute || delay > 61*time.Minute {
		t.Fatalf("retry backoff = %v, want about one hour", delay)
	}
	f.orch.taskQueue.Remove(retries[0])
	retry := retries[0]
	finished := retry.RunAt.Add(5 * time.Minute)
	retry.Op.Status = v1.OperationStatus_STATUS_SUCCESS
	retry.Op.UnixTimeEndMs = finished.UnixMilli()
	if err := f.log.Update(retry.Op); err != nil {
		t.Fatal(err)
	}
	f.orch.handleTaskCompletion(&retry, nil, nil)
	backups := queuedBackups(f.orch, "plan1")
	if len(backups) != 1 || !backups[0].RunAt.Equal(time.UnixMilli(finished.UnixMilli()).Add(12*time.Hour)) {
		t.Fatalf("expected one next backup after retry, got %v", backups)
	}
}

func queuedBackups(orch *Orchestrator, planID string) []stContainer {
	var backups []stContainer
	for _, queued := range orch.taskQueue.GetAll() {
		if queued.Task.Type() == "backup" && queued.Task.PlanID() == planID {
			backups = append(backups, queued)
		}
	}
	return backups
}

func TestClockJumpResetKeepsOtherPlansScheduled(t *testing.T) {
	f := newClockJumpBackupFixture(t)
	f.running.Op.Status = v1.OperationStatus_STATUS_INPROGRESS
	if err := f.log.Update(f.running.Op); err != nil {
		t.Fatal(err)
	}
	updated := proto.Clone(f.cfg).(*v1.Config)
	second := proto.Clone(updated.Plans[0]).(*v1.Plan)
	second.Id = "plan2"
	updated.Plans = append(updated.Plans, second)
	if err := f.orch.applyConfig(updated); err != nil {
		t.Fatal(err)
	}
	if got := len(queuedBackups(f.orch, "plan1")); got != 0 {
		t.Fatalf("active plan was duplicated: %d queued", got)
	}
	if got := len(queuedBackups(f.orch, "plan2")); got != 1 {
		t.Fatalf("other plan was not scheduled once: %d queued", got)
	}
	finished := f.now.Add(5 * time.Minute)
	f.running.Op.Status = v1.OperationStatus_STATUS_SUCCESS
	f.running.Op.UnixTimeEndMs = finished.UnixMilli()
	if err := f.log.Update(f.running.Op); err != nil {
		t.Fatal(err)
	}
	f.orch.handleTaskCompletion(&f.running, nil, nil)
	if got := len(queuedBackups(f.orch, "plan1")); got != 1 {
		t.Fatalf("completed plan was not scheduled once: %d queued", got)
	}
	if got := len(queuedBackups(f.orch, "plan2")); got != 1 {
		t.Fatalf("other plan changed after first plan completed: %d queued", got)
	}
	if err := f.orch.ScheduleDefaultTasks(updated); err != nil {
		t.Fatal(err)
	}
	for _, planID := range []string{"plan1", "plan2"} {
		if got := len(queuedBackups(f.orch, planID)); got != 1 {
			t.Fatalf("%s has %d backups after another reset, want one", planID, got)
		}
	}
}

func TestClockJumpUsesCurrentPlanAfterScheduleChange(t *testing.T) {
	f := newClockJumpBackupFixture(t)
	f.running.Op.Status = v1.OperationStatus_STATUS_INPROGRESS
	if err := f.log.Update(f.running.Op); err != nil {
		t.Fatal(err)
	}
	updated := proto.Clone(f.cfg).(*v1.Config)
	updated.Plans[0].Schedule.Schedule = &v1.Schedule_MaxFrequencyHours{MaxFrequencyHours: 24}
	if err := f.orch.applyConfig(updated); err != nil {
		t.Fatal(err)
	}
	if got := len(queuedBackups(f.orch, "plan1")); got != 0 {
		t.Fatalf("changed plan scheduled while old backup is active: %d queued", got)
	}
	finished := f.now.Add(5 * time.Minute)
	f.running.Op.Status = v1.OperationStatus_STATUS_SUCCESS
	f.running.Op.UnixTimeEndMs = finished.UnixMilli()
	if err := f.log.Update(f.running.Op); err != nil {
		t.Fatal(err)
	}
	f.orch.handleTaskCompletion(&f.running, nil, nil)
	backups := queuedBackups(f.orch, "plan1")
	if len(backups) != 1 || !backups[0].RunAt.Equal(finished.Add(24*time.Hour)) {
		t.Fatalf("expected one backup under updated 24h plan, got %v", backups)
	}
}

func TestClockJumpReschedulesAfterFailedBackup(t *testing.T) {
	f := newClockJumpBackupFixture(t)
	f.running.Op.Status = v1.OperationStatus_STATUS_INPROGRESS
	if err := f.log.Update(f.running.Op); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if err := f.orch.ScheduleDefaultTasks(f.cfg); err != nil {
			t.Fatal(err)
		}
		if got := len(queuedBackups(f.orch, "plan1")); got != 0 {
			t.Fatalf("reset %d duplicated active backup: %d queued", i, got)
		}
	}
	finished := f.now.Add(5 * time.Minute)
	f.running.Op.Status = v1.OperationStatus_STATUS_ERROR
	f.running.Op.UnixTimeEndMs = finished.UnixMilli()
	if err := f.log.Update(f.running.Op); err != nil {
		t.Fatal(err)
	}
	f.orch.handleTaskCompletion(&f.running, errors.New("backup failed"), nil)
	backups := queuedBackups(f.orch, "plan1")
	if len(backups) != 1 || !backups[0].RunAt.Equal(finished.Add(12*time.Hour)) {
		t.Fatalf("expected one next backup after failure, got %v", backups)
	}
}

func TestClockJumpConcurrentResetAndCompletion(t *testing.T) {
	for i := 0; i < 25; i++ {
		f := newClockJumpBackupFixture(t)
		finished := f.now.Add(5 * time.Minute)
		f.running.Op.Status = v1.OperationStatus_STATUS_SUCCESS
		f.running.Op.UnixTimeEndMs = finished.UnixMilli()
		if err := f.log.Update(f.running.Op); err != nil {
			t.Fatal(err)
		}
		done := make(chan error, 2)
		go func() { done <- f.orch.ScheduleDefaultTasks(f.cfg) }()
		go func() { f.orch.handleTaskCompletion(&f.running, nil, nil); done <- nil }()
		for j := 0; j < 2; j++ {
			select {
			case err := <-done:
				if err != nil {
					t.Fatal(err)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("queue reset and completion deadlocked")
			}
		}
		backups := queuedBackups(f.orch, "plan1")
		if len(backups) != 1 || !backups[0].RunAt.Equal(finished.Add(12*time.Hour)) {
			t.Fatalf("iteration %d: expected exactly one next backup, got %v", i, backups)
		}
	}
}

func TestClockJumpTwoActivePlansReset(t *testing.T) {
	f := newClockJumpBackupFixture(t)
	updated := proto.Clone(f.cfg).(*v1.Config)
	second := proto.Clone(updated.Plans[0]).(*v1.Plan)
	second.Id = "plan2"
	updated.Plans = append(updated.Plans, second)
	if err := f.orch.applyConfig(updated); err != nil {
		t.Fatal(err)
	}
	otherBackups := queuedBackups(f.orch, "plan2")
	if len(otherBackups) != 1 {
		t.Fatalf("expected second plan backup, got %d", len(otherBackups))
	}
	other := otherBackups[0]
	f.orch.taskQueue.Remove(other)
	for _, active := range []*stContainer{&f.running, &other} {
		active.Op.Status = v1.OperationStatus_STATUS_INPROGRESS
		if err := f.log.Update(active.Op); err != nil {
			t.Fatal(err)
		}
	}
	if err := f.orch.ScheduleDefaultTasks(updated); err != nil {
		t.Fatal(err)
	}
	for _, planID := range []string{"plan1", "plan2"} {
		if got := len(queuedBackups(f.orch, planID)); got != 0 {
			t.Fatalf("active %s was duplicated: %d queued", planID, got)
		}
	}
	f.running.Op.Status = v1.OperationStatus_STATUS_SUCCESS
	f.running.Op.UnixTimeEndMs = f.now.Add(5 * time.Minute).UnixMilli()
	other.Op.Status = v1.OperationStatus_STATUS_SUCCESS
	other.Op.UnixTimeEndMs = f.now.Add(6 * time.Minute).UnixMilli()
	for _, active := range []*stContainer{&f.running, &other} {
		if err := f.log.Update(active.Op); err != nil {
			t.Fatal(err)
		}
	}
	done := make(chan bool, 2)
	go func() { done <- f.orch.handleTaskCompletion(&f.running, nil, nil) }()
	go func() { done <- f.orch.handleTaskCompletion(&other, nil, nil) }()
	for i := 0; i < 2; i++ {
		select {
		case retried := <-done:
			if retried {
				t.Fatal("successful backup unexpectedly retried")
			}
		case <-time.After(2 * time.Second):
			t.Fatal("parallel plan completions deadlocked")
		}
	}
	for _, planID := range []string{"plan1", "plan2"} {
		if got := len(queuedBackups(f.orch, planID)); got != 1 {
			t.Fatalf("%s has %d next backups, want one", planID, got)
		}
	}
}
