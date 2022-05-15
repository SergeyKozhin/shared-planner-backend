package events

import (
	"context"
	"fmt"
	"time"

	"github.com/SergeyKozhin/shared-planner-backend/internal/model"
)

func (s *Service) UpdateEvent(ctx context.Context, id int64, ts time.Time, info *model.EventUpdate) error {
	oldEvent, err := s.eventsRepository.GetEventByID(ctx, s.db, id)
	if err != nil {
		return fmt.Errorf("get old event: %w", err)
	}

	diff := info.From.Sub(ts)
	from := oldEvent.From.Add(diff)
	to := from.Add(info.To.Sub(info.From))

	repeatRule := oldEvent.RepeatRule
	if oldEvent.RepeatType != model.RepeatTypeNone && !oldEvent.From.Equal(from) {
		var err error
		repeatRule, err = getRule(oldEvent.RepeatType, from, nil)
		if err != nil {
			return err
		}
	}

	exceptions := oldEvent.Exceptions
	if diff != 0 {
		newExceptions := make(map[int64]struct{}, len(oldEvent.Exceptions))
		for e := range oldEvent.Exceptions {
			newExceptions[time.Unix(e, 0).Add(diff).Unix()] = struct{}{}
		}

		exceptions = newExceptions
	}

	var endDate *time.Time
	if oldEvent.RepeatType == model.RepeatTypeNone {
		endDate = &to
	}

	if err := s.eventsRepository.UpdateEvent(ctx, s.db, &model.Event{
		ID:         oldEvent.ID,
		RepeatRule: repeatRule,
		Exceptions: exceptions,
		Until:      endDate,
		EventCreate: model.EventCreate{
			GroupID:       info.GroupID,
			EventType:     info.EventType,
			Title:         info.Title,
			Description:   info.Description,
			AllDay:        info.AllDay,
			From:          from,
			To:            to,
			RepeatType:    oldEvent.RepeatType,
			Notifications: info.Notifications,
			Attachments:   oldEvent.Attachments,
		},
	}); err != nil {
		return fmt.Errorf("eventsRepository.UpdateEvent: %w", err)
	}

	return nil
}

func (s *Service) UpdateEventInstance(ctx context.Context, id int64, ts time.Time, info *model.EventUpdate) error {
	oldEvent, err := s.eventsRepository.GetEventByID(ctx, s.db, id)
	if err != nil {
		return fmt.Errorf("get old event: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx")
	}
	defer tx.Rollback(ctx)

	oldEvent.Exceptions[ts.Unix()] = struct{}{}
	if err := s.eventsRepository.UpdateEvent(ctx, tx, &model.Event{
		ID:          oldEvent.ID,
		RepeatRule:  oldEvent.RepeatRule,
		Exceptions:  oldEvent.Exceptions,
		Until:       oldEvent.Until,
		EventCreate: oldEvent.EventCreate,
	}); err != nil {
		return fmt.Errorf("eventsRepository.UpdateEvent: %w", err)
	}

	if _, err := s.eventsRepository.CreateEvent(ctx, tx, &model.Event{
		RepeatRule: "",
		Exceptions: map[int64]struct{}{},
		Until:      &info.To,
		EventCreate: model.EventCreate{
			GroupID:       info.GroupID,
			EventType:     info.EventType,
			Title:         info.Title,
			Description:   info.Description,
			AllDay:        info.AllDay,
			From:          info.From,
			To:            info.To,
			RepeatType:    model.RepeatTypeNone,
			Notifications: info.Notifications,
			Attachments:   oldEvent.Attachments,
		},
	}); err != nil {
		return fmt.Errorf("eventsRepository.CreateEvent: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx")
	}

	return nil
}
