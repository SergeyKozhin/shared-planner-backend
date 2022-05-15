package notifications

import (
	"fmt"
	"time"
)

type durationValue int

const (
	durationValue5Minutes durationValue = iota
	durationValue10Minutes
	durationValue15Minutes
	durationValue30Minutes
	durationValueHour
	durationValueDay
)

func mapToDurationValue(d time.Duration) (durationValue, error) {
	var val durationValue
	switch time.Duration(d) {
	case 5 * time.Minute:
		val = durationValue5Minutes
	case 10 * time.Minute:
		val = durationValue10Minutes
	case 15 * time.Minute:
		val = durationValue15Minutes
	case 30 * time.Minute:
		val = durationValue30Minutes
	case 1 * time.Hour:
		val = durationValueHour
	case 24 * time.Hour:
		val = durationValueDay
	default:
		return 0, fmt.Errorf("unsupported duration: %v", d)
	}

	return val, nil
}
