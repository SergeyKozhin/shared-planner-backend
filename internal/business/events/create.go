package events

import (
	"context"
	"fmt"
	"time"

	"github.com/SergeyKozhin/shared-planner-backend/internal/model"
)

func (s *Service) CreateEvent(ctx context.Context, info *model.EventCreate) (*model.Event, error) {
	repeatRule := ""
	if info.RepeatType != model.RepeatTypeNone {
		var err error
		repeatRule, err = getRule(info.RepeatType, info.From, nil)
		if err != nil {
			return nil, err
		}
	}

	var endDate *time.Time
	if info.RepeatType == model.RepeatTypeNone {
		endDate = &info.To
	}

	event := &model.Event{
		RepeatRule:  repeatRule,
		Exceptions:  map[int64]struct{}{},
		Until:       endDate,
		EventCreate: *info,
	}

	id, err := s.eventsRepository.CreateEvent(ctx, s.db, event)
	if err != nil {
		return nil, fmt.Errorf("eventsRepository.CreateEvent: %w", err)
	}

	event.ID = fmt.Sprintf("%v_%v", id, info.From.Unix())
	return event, nil
}
