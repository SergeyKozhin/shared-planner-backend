package events

import (
	"context"
	"fmt"
	"time"

	"github.com/SergeyKozhin/shared-planner-backend/internal/model"
)

func (s *Service) DeleteEvent(ctx context.Context, id int64) error {
	if err := s.eventsRepository.DeleteEvent(ctx, s.db, id); err != nil {
		return fmt.Errorf("eventsRepository.DeleteEvent: %w", err)
	}

	return nil
}

func (s *Service) DeleteEventInstance(ctx context.Context, id int64, ts time.Time) error {
	oldEvent, err := s.eventsRepository.GetEventByID(ctx, s.db, id)
	if err != nil {
		return fmt.Errorf("get old event: %w", err)
	}

	oldEvent.Exceptions[ts.Unix()] = struct{}{}
	if err := s.eventsRepository.UpdateEvent(ctx, s.db, &model.Event{
		ID:          oldEvent.ID,
		RepeatRule:  oldEvent.RepeatRule,
		Exceptions:  oldEvent.Exceptions,
		Until:       oldEvent.Until,
		EventCreate: oldEvent.EventCreate,
	}); err != nil {
		return fmt.Errorf("eventsRepository.UpdateEvent: %w", err)
	}

	return nil
}
