//go:build tray

package main

import (
	"fmt"
	"sync/atomic"
	"time"

	"fyne.io/systray"
	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/oplog"
)

// debounceWindow is how long the refresh loop waits after an oplog event to let
// a burst of progress events settle before recomputing the status.
const debounceWindow = 1 * time.Second

// trayState is the overall backup state reflected by the tray icon.
type trayState int

const (
	stateIdle    trayState = iota // no backups recorded yet
	stateRunning                  // a backup is currently in progress
	stateOK                       // most recent backup succeeded
	stateWarning                  // most recent backup completed with warnings
	stateError                    // most recent backup failed
)

// trayStatus reflects backup state from the oplog in the tray icon and tooltip
// (addresses #894). Oplog events are coalesced through notify so a burst of
// progress updates triggers at most one recompute per debounce window.
type trayStatus struct {
	log    atomic.Pointer[oplog.OpLog]
	notify chan struct{}
}

func newTrayStatus() *trayStatus {
	return &trayStatus{notify: make(chan struct{}, 1)}
}

// attach is invoked via the onOpLogReady hook once the oplog exists. It
// subscribes for changes and requests an initial refresh.
func (t *trayStatus) attach(log *oplog.OpLog) {
	t.log.Store(log)
	sub := oplog.Subscription(func(_ []*v1.Operation, _ oplog.OperationEvent) { t.poke() })
	log.Subscribe(oplog.Query{}, &sub)
	t.poke()
}

// poke requests a refresh. The buffered channel drops the signal if one is
// already pending, collapsing bursts into a single wake-up.
func (t *trayStatus) poke() {
	select {
	case t.notify <- struct{}{}:
	default:
	}
}

// run coalesces oplog events into at most one refresh per debounce window. Start
// it once the systray is live so icon and tooltip writes take effect.
func (t *trayStatus) run() {
	for range t.notify {
		t.refresh()
		time.Sleep(debounceWindow)
	}
}

// refresh recomputes the status and writes it to the tray.
func (t *trayStatus) refresh() {
	log := t.log.Load()
	if log == nil {
		return
	}
	state, tooltip := computeStatus(log)
	if ic := statusIcon(state); ic != nil {
		systray.SetIcon(ic)
	}
	systray.SetTooltip(tooltip)
}

// computeStatus returns the tray state and tooltip for the most recent backup.
// It scans the oplog newest-first for the first backup that is not a scheduled
// future run (the orchestrator pre-creates a PENDING op for the next run), then
// maps that op's status to an icon. Only backup operations count —
// forget/prune/check/etc. never move the icon.
func computeStatus(log *oplog.OpLog) (trayState, string) {
	var last *v1.Operation
	_ = log.Query(oplog.Query{Reversed: true}, func(op *v1.Operation) error {
		// Match on the oneof type rather than GetOperationBackup(), which is nil
		// when the inner message is unset on an otherwise-backup op.
		if _, ok := op.GetOp().(*v1.Operation_OperationBackup); !ok {
			return nil
		}
		if op.GetStatus() == v1.OperationStatus_STATUS_PENDING {
			return nil // scheduled future run; not a result and not in progress
		}
		last = op
		return oplog.ErrStopIteration
	})

	if last == nil {
		return stateIdle, "Backrest — no backups yet"
	}
	when := relativeTime(last.GetUnixTimeEndMs())
	switch last.GetStatus() {
	case v1.OperationStatus_STATUS_INPROGRESS:
		return stateRunning, "Backrest — backup in progress…"
	case v1.OperationStatus_STATUS_SUCCESS:
		return stateOK, "Backrest — last backup succeeded " + when
	case v1.OperationStatus_STATUS_WARNING:
		return stateWarning, "Backrest — last backup finished with warnings " + when
	case v1.OperationStatus_STATUS_ERROR, v1.OperationStatus_STATUS_SYSTEM_CANCELLED:
		return stateError, "Backrest — last backup failed " + when
	case v1.OperationStatus_STATUS_USER_CANCELLED:
		return stateIdle, "Backrest — last backup was cancelled " + when
	default:
		return stateIdle, "Backrest — no backups yet"
	}
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
