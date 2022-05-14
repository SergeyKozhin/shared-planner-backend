package events

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/SergeyKozhin/shared-planner-backend/internal/model"
	"github.com/teambition/rrule-go"
)

func (s *Service) GetEventByID(ctx context.Context, id int64, ts time.Time) (*model.Event, error) {
	event, err := s.eventsRepository.GetEventByID(ctx, s.db, id)
	if err != nil {
		return nil, fmt.Errorf("eventsRepository.GetEventByID: %w", err)
	}

	if event.RepeatType == model.RepeatTypeNone {
		if !event.From.Equal(ts) {
			return nil, model.ErrNoRecord
		}
		return &model.Event{
			ID:          fmt.Sprintf("%v_%v", event.ID, event.From.Unix()),
			EventCreate: event.EventCreate,
		}, err
	}

	rOption, err := rrule.StrToROption(event.RepeatRule)
	if err != nil {
		return nil, fmt.Errorf("parse repeat rule %q: %w", event.RepeatRule, err)
	}
	rule, err := rrule.NewRRule(*rOption)
	if err != nil {
		return nil, fmt.Errorf("make rule: %w", err)
	}

	if !rule.After(ts, true).Equal(ts) {
		return nil, model.ErrNoRecord
	}

	if _, ok := event.Exceptions[ts.Unix()]; ok {
		return nil, model.ErrNoRecord
	}

	duration := event.To.Sub(event.From)
	return &model.Event{
		ID:         fmt.Sprintf("%v_%v", event.ID, ts.Unix()),
		RepeatRule: event.RepeatRule,
		Exceptions: event.Exceptions,
		EventCreate: model.EventCreate{
			GroupID:       event.GroupID,
			EventType:     event.EventType,
			Title:         event.Title,
			Description:   event.Description,
			AllDay:        event.AllDay,
			From:          ts,
			To:            ts.Add(duration),
			RepeatType:    event.RepeatType,
			Notifications: event.Notifications,
			Attachments:   event.Attachments,
		},
	}, nil
}

func (s *Service) GetEvents(ctx context.Context, filter model.EventsFilter) ([]*model.Event, error) {
	baseEvents, err := s.eventsRepository.GetEvents(ctx, s.db, filter)
	if err != nil {
		return nil, fmt.Errorf("eventsRepository.GetEvents: %w", err)
	}

	var res []*model.Event

	for _, e := range baseEvents {
		if e.RepeatType == model.RepeatTypeNone {
			res = append(res, &model.Event{
				ID:          fmt.Sprintf("%v_%v", e.ID, e.From.Unix()),
				EventCreate: e.EventCreate,
			})
			continue
		}

		duration := e.To.Sub(e.From)

		rOption, err := rrule.StrToROption(e.RepeatRule)
		if err != nil {
			return nil, fmt.Errorf("parse repeat rule %q: %w", e.RepeatRule, err)
		}
		rule, err := rrule.NewRRule(*rOption)
		if err != nil {
			return nil, fmt.Errorf("make rule: %w", err)
		}

		repeats := rule.Between(e.From, filter.To.Add(-1), true)
		for _, r := range repeats {
			from := r
			to := r.Add(duration)

			if filter.To.Before(from) || to.Before(filter.From) {
				continue
			}

			if _, ok := e.Exceptions[r.Unix()]; ok {
				continue
			}

			res = append(res, &model.Event{
				ID:         fmt.Sprintf("%v_%v", e.ID, from.Unix()),
				RepeatRule: e.RepeatRule,
				Exceptions: e.Exceptions,
				EventCreate: model.EventCreate{
					GroupID:       e.GroupID,
					EventType:     e.EventType,
					Title:         e.Title,
					Description:   e.Description,
					AllDay:        e.AllDay,
					From:          from,
					To:            to,
					RepeatType:    e.RepeatType,
					Notifications: e.Notifications,
					Attachments:   e.Attachments,
				},
			})
		}
	}

	sort.SliceStable(res, func(i, j int) bool {
		return res[i].From.Before(res[j].From)
	})

	return res, nil
}
