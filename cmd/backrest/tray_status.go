//go:build tray

package main

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/systray"
	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/oplog"
)

// trayState is the overall backup state reflected by the tray icon.
type trayState int

const (
	stateIdle    trayState = iota // no backups recorded yet
	stateRunning                  // a backup is currently in progress
	stateOK                       // most recent backup succeeded
	stateWarning                  // most recent backup completed with warnings
	stateError                    // most recent backup failed
)

// errStopQuery short-circuits an oplog query once the wanted op is found.
var errStopQuery = errors.New("stop")

// trayStatus tracks backup state from the oplog and updates the tray icon and
// tooltip to match (addresses #894). Updates are coalesced so a burst of
// progress events triggers at most one recompute per window.
type trayStatus struct {
	mu      sync.Mutex
	log     *oplog.OpLog
	cur     trayState
	tooltip string
	ready   atomic.Bool
	pending atomic.Bool
}

func newTrayStatus() *trayStatus {
	return &trayStatus{cur: stateIdle, tooltip: "Backrest"}
}

// attach is invoked via the onOpLogReady hook once the oplog exists. It seeds
// the initial state and subscribes for subsequent changes.
func (t *trayStatus) attach(log *oplog.OpLog) {
	t.mu.Lock()
	t.log = log
	t.mu.Unlock()

	sub := oplog.Subscription(func(_ []*v1.Operation, _ oplog.OperationEvent) {
		t.scheduleRefresh()
	})
	log.Subscribe(oplog.Query{}, &sub)
	t.doRefresh()
}

// markReady is called once the systray is live so icon/tooltip writes land.
func (t *trayStatus) markReady() {
	t.ready.Store(true)
	t.apply()
}

// scheduleRefresh coalesces a burst of oplog events into one recompute.
func (t *trayStatus) scheduleRefresh() {
	if t.pending.Swap(true) {
		return
	}
	go func() {
		time.Sleep(750 * time.Millisecond)
		t.pending.Store(false)
		t.doRefresh()
	}()
}

func (t *trayStatus) doRefresh() {
	t.mu.Lock()
	log := t.log
	t.mu.Unlock()
	if log == nil {
		return
	}

	state := stateIdle
	tooltip := "Backrest — no backups yet"
	running := false
	var last *v1.Operation

	// Reversed = newest first. Skip non-backup ops; a running backup flips the
	// state, otherwise the first completed backup is the most recent result.
	_ = log.Query(oplog.Query{Reversed: true}, func(op *v1.Operation) error {
		// Only backup operations drive the headline status. Match on the oneof
		// type rather than GetOperationBackup(), which is nil when the inner
		// message is unset on an otherwise-backup op.
		if _, ok := op.GetOp().(*v1.Operation_OperationBackup); !ok {
			return nil
		}
		switch op.GetStatus() {
		case v1.OperationStatus_STATUS_INPROGRESS, v1.OperationStatus_STATUS_PENDING:
			running = true
			return nil
		default:
			last = op
			return errStopQuery
		}
	})

	switch {
	case running:
		state = stateRunning
		tooltip = "Backrest — backup in progress…"
	case last != nil:
		when := relativeTime(last.GetUnixTimeEndMs())
		switch last.GetStatus() {
		case v1.OperationStatus_STATUS_SUCCESS:
			state, tooltip = stateOK, "Backrest — last backup succeeded "+when
		case v1.OperationStatus_STATUS_WARNING:
			state, tooltip = stateWarning, "Backrest — last backup finished with warnings "+when
		case v1.OperationStatus_STATUS_ERROR, v1.OperationStatus_STATUS_SYSTEM_CANCELLED:
			state, tooltip = stateError, "Backrest — last backup failed "+when
		case v1.OperationStatus_STATUS_USER_CANCELLED:
			state, tooltip = stateIdle, "Backrest — last backup was cancelled "+when
		}
	}

	t.mu.Lock()
	changed := state != t.cur || tooltip != t.tooltip
	t.cur, t.tooltip = state, tooltip
	t.mu.Unlock()
	if changed {
		t.apply()
	}
}

func (t *trayStatus) apply() {
	if !t.ready.Load() {
		return
	}
	t.mu.Lock()
	state, tooltip := t.cur, t.tooltip
	t.mu.Unlock()

	if ic := statusIcon(state); ic != nil {
		systray.SetIcon(ic)
	}
	systray.SetTooltip(tooltip)
}

func relativeTime(unixMs int64) string {
	if unixMs == 0 {
		return ""
	}
	d := time.Since(time.UnixMilli(unixMs))
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}
