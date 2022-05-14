package events

import (
	"fmt"
	"time"

	"github.com/SergeyKozhin/shared-planner-backend/internal/model"
	"github.com/teambition/rrule-go"
)

func getRule(t model.RepeatType, from time.Time, to *time.Time) (string, error) {
	var freq rrule.Frequency
	var interval int

	switch t {
	case model.RepeatTypeNone:
		return "", nil
	case model.RepeatTypeEveryDay:
		freq = rrule.DAILY
		interval = 1
	case model.RepeatTypeEveryThreeDays:
		freq = rrule.DAILY
		interval = 3
	case model.RepeatTypeEveryWeek:
		freq = rrule.WEEKLY
		interval = 1
	case model.RepeatTypeEveryMonth:
		freq = rrule.MONTHLY
		interval = 1
	case model.RepeatTypeEveryYear:
		freq = rrule.YEARLY
		interval = 1
	default:
		return "", fmt.Errorf("unknown repeat type: %v", t)
	}

	opt := rrule.ROption{
		Freq:     freq,
		Interval: interval,
		Dtstart:  from.UTC(),
	}

	if to != nil {
		opt.Until = *to
	}

	rule, err := rrule.NewRRule(opt)
	if err != nil {
		return "", fmt.Errorf("creating rule: %w", err)
	}

	return rule.String(), nil
}
