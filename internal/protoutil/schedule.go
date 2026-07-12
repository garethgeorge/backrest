package protoutil

import (
	"errors"
	"fmt"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/gitploy-io/cronexpr"
)

var ErrScheduleDisabled = errors.New("never")

// ScheduleEnabled returns true if the schedule is set and will actually fire,
// i.e. it is non-nil and its oneof case is not `disabled`. Use this (rather
// than a nil check) anywhere behavior branches on whether a scheduled policy
// is active: the WebUI serializes a {disabled: true} schedule for policies
// the user has turned off, so a non-nil schedule may still never run.
func ScheduleEnabled(sched *v1.Schedule) bool {
	if sched == nil {
		return false
	}
	switch sched.GetSchedule().(type) {
	case *v1.Schedule_Disabled, nil:
		return false
	default:
		return true
	}
}

// ResolveSchedule resolves a schedule to the next time it should run based on last execution.
// note that this is different from backup behavior which is always relative to the current time.
func ResolveSchedule(sched *v1.Schedule, lastRan time.Time, curTime time.Time) (time.Time, error) {
	var t time.Time
	switch sched.GetClock() {
	case v1.Schedule_CLOCK_DEFAULT, v1.Schedule_CLOCK_LOCAL:
		t = curTime.Local()
	case v1.Schedule_CLOCK_UTC:
		t = curTime.UTC()
	case v1.Schedule_CLOCK_LAST_RUN_TIME:
		t = lastRan
	default:
		return time.Time{}, fmt.Errorf("unknown clock type: %v", sched.GetClock().String())
	}

	switch s := sched.GetSchedule().(type) {
	case *v1.Schedule_Disabled, nil:
		return time.Time{}, ErrScheduleDisabled
	case *v1.Schedule_MaxFrequencyDays:
		return t.Add(time.Duration(s.MaxFrequencyDays) * 24 * time.Hour), nil
	case *v1.Schedule_MaxFrequencyHours:
		return t.Add(time.Duration(s.MaxFrequencyHours) * time.Hour), nil
	case *v1.Schedule_Cron:
		cron, err := cronexpr.ParseInLocation(s.Cron, time.Now().Location().String())
		if err != nil {
			return time.Time{}, fmt.Errorf("parse cron %q: %w", s.Cron, err)
		}
		if cron.Next(t).IsZero() || cron.Next(t).Before(t) {
			return time.Time{}, fmt.Errorf("cron %q may be malformed, next scheduled time is in the past %v", s.Cron, t)
		}
		return cron.Next(t), nil
	default:
		return time.Time{}, fmt.Errorf("unknown schedule type: %T", s)
	}
}

// cronPeriodSamples is the number of consecutive fire-to-fire gaps sampled to
// estimate a cron schedule's nominal period; enough to span irregular patterns
// like weekdays-only (whose widest gap is the weekend).
const cronPeriodSamples = 8

// NominalPeriod returns the largest expected gap between consecutive runs of the
// schedule. It is a display-grade staleness bound, not a scheduling calculation.
func NominalPeriod(sched *v1.Schedule, from time.Time) (time.Duration, error) {
	switch s := sched.GetSchedule().(type) {
	case *v1.Schedule_Disabled, nil:
		return 0, ErrScheduleDisabled
	case *v1.Schedule_MaxFrequencyDays:
		return time.Duration(s.MaxFrequencyDays) * 24 * time.Hour, nil
	case *v1.Schedule_MaxFrequencyHours:
		return time.Duration(s.MaxFrequencyHours) * time.Hour, nil
	case *v1.Schedule_Cron:
		cron, err := cronexpr.ParseInLocation(s.Cron, from.Location().String())
		if err != nil {
			return 0, fmt.Errorf("parse cron %q: %w", s.Cron, err)
		}
		var maxGap time.Duration
		t := cron.Next(from)
		for i := 0; i < cronPeriodSamples; i++ {
			next := cron.Next(t)
			if next.IsZero() || !next.After(t) {
				return 0, fmt.Errorf("cron %q: could not compute consecutive run times", s.Cron)
			}
			if gap := next.Sub(t); gap > maxGap {
				maxGap = gap
			}
			t = next
		}
		return maxGap, nil
	default:
		return 0, fmt.Errorf("unknown schedule type: %T", s)
	}
}

func ValidateSchedule(sched *v1.Schedule) error {
	switch s := sched.GetSchedule().(type) {
	case *v1.Schedule_MaxFrequencyDays:
		if s.MaxFrequencyDays < 1 {
			return errors.New("invalid max frequency days")
		}
	case *v1.Schedule_MaxFrequencyHours:
		if s.MaxFrequencyHours < 1 {
			return errors.New("invalid max frequency hours")
		}
	case *v1.Schedule_Cron:
		if s.Cron == "" {
			return errors.New("empty cron expression")
		}
		cron, err := cronexpr.ParseInLocation(s.Cron, time.Now().Location().String())
		if err != nil {
			return fmt.Errorf("invalid cron %q: %w", s.Cron, err)
		}
		if next := cron.Next(time.Now()); next.IsZero() || next.Year() < 2000 {
			return fmt.Errorf("invalid cron %q: next scheduled time is invalid (check for DOW=7 usage)", s.Cron)
		}
	case nil:
		return nil
	case *v1.Schedule_Disabled:
		if !s.Disabled {
			return errors.New("disabled boolean must be set to true")
		}
	default:
		return fmt.Errorf("unknown schedule type: %T", s)
	}
	return nil
}
