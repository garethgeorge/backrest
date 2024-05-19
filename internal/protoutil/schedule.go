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
func ResolveSchedule(sched *v1.Schedule, lastRan time.Time) (time.Time, error) {
	switch s := sched.GetSchedule().(type) {
	case *v1.Schedule_Disabled:
		return time.Time{}, ErrScheduleDisabled
	case *v1.Schedule_MaxFrequencyDays:
		return lastRan.Add(time.Duration(s.MaxFrequencyDays) * 24 * time.Hour), nil
	case *v1.Schedule_MaxFrequencyHours:
		return lastRan.Add(time.Duration(s.MaxFrequencyHours) * time.Hour), nil
	case *v1.Schedule_Cron:
		cron, err := cronexpr.ParseInLocation(s.Cron, time.Now().Location().String())
		if err != nil {
			return time.Time{}, fmt.Errorf("parse cron %q: %w", s.Cron, err)
		}
		return cron.Next(lastRan), nil
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
		_, err := cronexpr.ParseInLocation(s.Cron, time.Now().Location().String())
		if err != nil {
			return fmt.Errorf("invalid cron %q: %w", s.Cron, err)
		}
	case *v1.Schedule_Disabled:
		if !s.Disabled {
			return errors.New("disabled boolean must be set to true")
		}
	default:
		return fmt.Errorf("unknown schedule type: %T", s)
	}
	return nil
}
