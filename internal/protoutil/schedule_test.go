package protoutil_test

import (
	"testing"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/protoutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveSchedule(t *testing.T) {
	// Use Local time to match ResolveSchedule's ParseInLocation(time.Now().Location())
	// ensuring cron expression and reference time are in same zone.
	now := time.Date(2023, 10, 1, 10, 0, 0, 0, time.Local) // Sunday, Oct 1st 2023 10:00 Local

	tests := []struct {
		name        string
		schedule    *v1.Schedule
		lastRan     time.Time
		curTime     time.Time
		expected    time.Time
		expectError bool
	}{
		{
			name: "MaxFrequencyDays - 1 day",
			schedule: &v1.Schedule{
				Clock: v1.Schedule_CLOCK_LOCAL,
				Schedule: &v1.Schedule_MaxFrequencyDays{
					MaxFrequencyDays: 1,
				},
			},
			lastRan:  now.Add(-24 * time.Hour),
			curTime:  now,
			expected: now.Add(24 * time.Hour),
		},
		{
			name: "Cron - Every minute",
			schedule: &v1.Schedule{
				Clock: v1.Schedule_CLOCK_LOCAL,
				Schedule: &v1.Schedule_Cron{
					Cron: "* * * * *",
				},
			},
			lastRan:  now,
			curTime:  now,
			expected: now.Add(1 * time.Minute),
		},
		{
			name: "Cron - Sunday (0) - Should work",
			schedule: &v1.Schedule{
				Clock: v1.Schedule_CLOCK_LOCAL,
				Schedule: &v1.Schedule_Cron{
					Cron: "0 10 * * 0", // 10:00 AM on Sunday
				},
			},
			lastRan: now.Add(-1 * time.Hour),
			curTime: now,
			// now is Sunday 10:00:00.
			// If we are exactly at scheduled time, Next() usually returns next slot?
			// Let's assume next slot is next week.
			expected: now.Add(7 * 24 * time.Hour),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := protoutil.ResolveSchedule(tt.schedule, tt.lastRan, tt.curTime)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.WithinDuration(t, tt.expected, got, 1*time.Second)
			}
		})
	}
}

func TestValidateSchedule(t *testing.T) {
	tests := []struct {
		name          string
		schedule      *v1.Schedule
		expectError   bool
		errorContains string
	}{
		{
			name: "Valid Cron (0)",
			schedule: &v1.Schedule{
				Schedule: &v1.Schedule_Cron{Cron: "0 10 * * 0"},
			},
			expectError: false,
		},
		{
			name: "Invalid Cron (7) - Validation Error",
			schedule: &v1.Schedule{
				Schedule: &v1.Schedule_Cron{Cron: "0 10 * * 7"},
			},
			expectError:   true,
			errorContains: "check for DOW=7 usage",
		},
		{
			name: "Valid Frequency",
			schedule: &v1.Schedule{
				Schedule: &v1.Schedule_MaxFrequencyDays{MaxFrequencyDays: 1},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := protoutil.ValidateSchedule(tt.schedule)
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func transactionalTimeEqual(t1, t2 time.Time) bool {
	return t1.Equal(t2)
}
