//go:build tray

package main

import (
	"testing"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/oplog/memstore"
)

func newTestOpLog(t *testing.T) *oplog.OpLog {
	t.Helper()
	log, err := oplog.NewOpLog(memstore.NewMemStore())
	if err != nil {
		t.Fatalf("NewOpLog: %v", err)
	}
	return log
}

var opStartTime int64 = 1_000

func backupOp(status v1.OperationStatus) *v1.Operation {
	opStartTime += 1_000
	return &v1.Operation{
		InstanceId:      "test-instance",
		RepoId:          "test-repo",
		RepoGuid:        "test-repo",
		PlanId:          "test-plan",
		UnixTimeStartMs: opStartTime,
		UnixTimeEndMs:   opStartTime + 500,
		Status:          status,
		Op:              &v1.Operation_OperationBackup{OperationBackup: &v1.OperationBackup{}},
	}
}

func nonBackupOp(status v1.OperationStatus) *v1.Operation {
	opStartTime += 1_000
	return &v1.Operation{
		InstanceId:      "test-instance",
		RepoId:          "test-repo",
		RepoGuid:        "test-repo",
		PlanId:          "test-plan",
		UnixTimeStartMs: opStartTime,
		UnixTimeEndMs:   opStartTime + 500,
		Status:          status,
		Op:              &v1.Operation_OperationForget{OperationForget: &v1.OperationForget{}},
	}
}

func TestTrayStatus(t *testing.T) {
	log := newTestOpLog(t)
	ts := newTrayStatus()
	ts.attach(log)

	// No operations yet.
	ts.doRefresh()
	if ts.cur != stateIdle {
		t.Fatalf("empty oplog: got %v, want stateIdle", ts.cur)
	}

	steps := []struct {
		name string
		op   *v1.Operation
		want trayState
	}{
		{"successful backup", backupOp(v1.OperationStatus_STATUS_SUCCESS), stateOK},
		{"failed backup is newest", backupOp(v1.OperationStatus_STATUS_ERROR), stateError},
		{"non-backup success is ignored", nonBackupOp(v1.OperationStatus_STATUS_SUCCESS), stateError},
		{"warning backup is newest", backupOp(v1.OperationStatus_STATUS_WARNING), stateWarning},
		{"in-progress backup shows running", backupOp(v1.OperationStatus_STATUS_INPROGRESS), stateRunning},
		{"completed success after run", backupOp(v1.OperationStatus_STATUS_SUCCESS), stateOK},
	}

	for _, s := range steps {
		if err := log.Add(s.op); err != nil {
			t.Fatalf("%s: Add: %v", s.name, err)
		}
		ts.doRefresh()
		if ts.cur != s.want {
			t.Errorf("%s: got state %v, want %v", s.name, ts.cur, s.want)
		}
	}
}

func TestTrayStatusInProgressOverridesOlderResult(t *testing.T) {
	log := newTestOpLog(t)
	ts := newTrayStatus()
	ts.attach(log)

	if err := log.Add(backupOp(v1.OperationStatus_STATUS_ERROR)); err != nil {
		t.Fatal(err)
	}
	if err := log.Add(backupOp(v1.OperationStatus_STATUS_INPROGRESS)); err != nil {
		t.Fatal(err)
	}
	ts.doRefresh()
	if ts.cur != stateRunning {
		t.Errorf("in-progress over older error: got %v, want stateRunning", ts.cur)
	}
}
