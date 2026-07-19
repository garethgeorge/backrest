package api

import (
	"testing"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
)

func TestSummaryIndexedSnapshotFallback(t *testing.T) {
	day := func(n int) time.Time {
		return time.Date(2026, 7, 16+n, 0, 0, 0, 0, time.Local)
	}
	atNoon := func(n int) time.Time { return day(n).Add(12 * time.Hour) }

	acc := newSummaryAcc(day(0))
	acc.observeBackup(&v1.Operation{
		SnapshotId:      "existing-backup",
		Status:          v1.OperationStatus_STATUS_SUCCESS,
		UnixTimeStartMs: atNoon(1).UnixMilli(),
		UnixTimeEndMs:   atNoon(1).Add(time.Minute).UnixMilli(),
	}, &v1.OperationBackup{})
	acc.observeIndexedSnapshot(&v1.OperationIndexSnapshot{
		Snapshot: &v1.ResticSnapshot{
			Id:         "existing-backup",
			UnixTimeMs: atNoon(1).UnixMilli(),
		},
	})
	acc.observeIndexedSnapshot(&v1.OperationIndexSnapshot{
		Snapshot: &v1.ResticSnapshot{
			Id:         "fallback-snapshot",
			UnixTimeMs: atNoon(0).UnixMilli(),
			Summary: &v1.SnapshotSummary{
				DataAdded:           25,
				TotalBytesProcessed: 250,
			},
		},
	})
	acc.observeIndexedSnapshot(&v1.OperationIndexSnapshot{
		Forgot: true,
		Snapshot: &v1.ResticSnapshot{
			Id:         "forgotten-snapshot",
			UnixTimeMs: atNoon(2).UnixMilli(),
		},
	})

	history := acc.finalize("test", atNoon(2), 0).HistoryLast_30Days
	statusCount := func(dayIndex int, status v1.OperationStatus) int64 {
		for _, sc := range history[dayIndex].StatusCounts {
			if sc.Status == status {
				return sc.Count
			}
		}
		return 0
	}

	if got := statusCount(0, v1.OperationStatus_STATUS_SUCCESS); got != 1 {
		t.Errorf("fallback day success count = %d, want 1", got)
	}
	if history[0].BytesAdded != 25 || history[0].BytesScanned != 250 {
		t.Errorf("fallback day bytes = (%d, %d), want (25, 250)",
			history[0].BytesAdded, history[0].BytesScanned)
	}
	if got := statusCount(1, v1.OperationStatus_STATUS_SUCCESS); got != 1 {
		t.Errorf("deduplicated day success count = %d, want 1", got)
	}
	if got := statusCount(2, v1.OperationStatus_STATUS_SUCCESS); got != 0 {
		t.Errorf("forgotten snapshot success count = %d, want 0", got)
	}
}

func TestSummaryOverdueFlags(t *testing.T) {
	// Fixed UTC reference: "today" is Jun 30 2026; the window is the prior 9 days.
	day := func(n int) time.Time { return time.Date(2026, 6, 21, 0, 0, 0, 0, time.UTC).AddDate(0, 0, n) }
	at := func(n int, hour int) time.Time { return day(n).Add(time.Duration(hour) * time.Hour) }
	now := at(9, 12) // noon on the last day

	tests := []struct {
		name                     string
		lastOkBackupBeforeWindow time.Time
		okBackupDates            []time.Time
		allowedGap               time.Duration
		expected                 []int // indices of days expected to be flagged overdue
	}{
		{
			name:          "no backups ever means no expectation",
			okBackupDates: nil,
			allowedGap:    24 * time.Hour,
			expected:      nil,
		},
		{
			name:          "daily cadence kept",
			okBackupDates: []time.Time{at(0, 1), at(1, 1), at(2, 1), at(3, 1), at(4, 1), at(5, 1), at(6, 1), at(7, 1), at(8, 1), at(9, 1)},
			allowedGap:    30 * time.Hour, // 24h * 1.25 grace
			expected:      nil,
		},
		{
			name:          "daily cadence with one skipped day",
			okBackupDates: []time.Time{at(0, 1), at(1, 1), at(2, 1), at(4, 1), at(5, 1), at(6, 1), at(7, 1), at(8, 1), at(9, 1)},
			allowedGap:    30 * time.Hour,
			// At day3's close the day2 01:00 backup is 47h old; day4 recovers with a
			// backup of its own (and renders as backed-up, not overdue).
			expected: []int{3},
		},
		{
			name:          "weekly cadence never overdue within window",
			okBackupDates: []time.Time{at(1, 2), at(8, 2)},
			allowedGap:    8 * 24 * time.Hour,
			expected:      nil,
		},
		{
			name:          "stalled plan flags tail days",
			okBackupDates: []time.Time{at(0, 1), at(1, 1)},
			allowedGap:    30 * time.Hour,
			// At day2's close the day1 01:00 backup is 47h old, and it only ages from there.
			expected: []int{2, 3, 4, 5, 6, 7, 8, 9},
		},
		{
			name:                     "pre-window staleness resolved in window stays muted",
			lastOkBackupBeforeWindow: day(-5),
			okBackupDates:            []time.Time{at(3, 1), at(5, 1), at(7, 1), at(9, 1)},
			allowedGap:               48 * time.Hour,
			// The stale day(-5) anchor is ignored because the window has its own good
			// backups: the overdue stretch ended before day3 and is resolved history.
			expected: nil,
		},
		{
			name:                     "fully stalled plan flags every day",
			lastOkBackupBeforeWindow: day(-5),
			okBackupDates:            nil,
			allowedGap:               48 * time.Hour,
			// No good backup in the window: the anchor drives staleness, and every
			// day closes more than 48h past it.
			expected: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
		},
		{
			name:          "sub-daily schedule with a backup each day is never overdue",
			okBackupDates: []time.Time{at(0, 2), at(1, 2), at(2, 2), at(3, 2), at(4, 2), at(5, 2), at(6, 2), at(7, 2), at(8, 2), at(9, 2)},
			allowedGap:    75 * time.Minute, // hourly cadence * 1.25 grace
			// Every day has an OK backup at 02:00. Its close (midnight) is ~22h past
			// that, far beyond the gap, but a day that backed up is never overdue.
			expected: nil,
		},
		{
			name:          "sub-daily schedule flags a fully missed day",
			okBackupDates: []time.Time{at(0, 2), at(1, 2), at(3, 2), at(4, 2), at(5, 2), at(6, 2), at(7, 2), at(8, 2), at(9, 2)}, // day2 missing
			allowedGap:    75 * time.Minute,
			// Day 2 has no OK backup of its own and closes far past the allowed gap;
			// day 3 recovers with its own backup and stays backed-up.
			expected: []int{2},
		},
		{
			name:          "gap exactly at allowed staleness is not overdue",
			okBackupDates: []time.Time{at(0, 0), at(2, 0), at(4, 0), at(6, 0), at(8, 0)},
			allowedGap:    48 * time.Hour,
			// Every day's close lands at most exactly on a deadline, never past one.
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			acc := newSummaryAcc(day(0))
			acc.reachedCutoff = true // span the full window from day 0
			acc.okBackupDates = tt.okBackupDates
			acc.lastOkBackupBeforeWindow = tt.lastOkBackupBeforeWindow

			history := acc.finalize("test", now, tt.allowedGap).HistoryLast_30Days
			if len(history) != 10 {
				t.Fatalf("expected 10 day buckets, got %d", len(history))
			}
			expectedSet := make(map[int]bool)
			for _, i := range tt.expected {
				expectedSet[i] = true
			}
			for i, bucket := range history {
				if bucket.Overdue != expectedSet[i] {
					t.Errorf("day %d: overdue = %v, want %v", i, bucket.Overdue, expectedSet[i])
				}
			}
		})
	}
}
