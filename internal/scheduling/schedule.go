package scheduling

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

type Schedule struct {
	spec     string
	schedule cron.Schedule
}

func ParseSchedule(spec string) (Schedule, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return Schedule{}, errors.New("schedule is required")
	}

	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	s, err := parser.Parse(spec)
	if err != nil {
		return Schedule{}, fmt.Errorf("invalid schedule %q: %w", spec, err)
	}
	return Schedule{spec: spec, schedule: s}, nil
}

func (s Schedule) Spec() string { return s.spec }

func (s Schedule) Next(from time.Time, loc *time.Location) time.Time {
	if s.schedule == nil {
		return time.Time{}
	}
	if loc == nil {
		loc = time.UTC
	}
	next := s.schedule.Next(from.In(loc))
	return next.UTC()
}
