package protoutil

import (
	"errors"
	"fmt"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/gitploy-io/cronexpr"
)

var ErrScheduleDisabled = errors.New("never")

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
