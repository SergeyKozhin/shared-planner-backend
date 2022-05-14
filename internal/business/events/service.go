package events

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/SergeyKozhin/shared-planner-backend/internal/database"
	"github.com/SergeyKozhin/shared-planner-backend/internal/model"
	"github.com/teambition/rrule-go"
)

type Service struct {
	db               database.PGX
	eventsRepository eventsRepository
}

type eventsRepository interface {
	CreateEvent(ctx context.Context, q database.Queryable, event *model.Event) (int64, error)
	GetEventByID(ctx context.Context, q database.Queryable, id int64) (*model.Event, error)
	GetEvents(ctx context.Context, q database.Queryable, filter model.EventsFilter) ([]*model.Event, error)
}

func NewService(db database.PGX, repo eventsRepository) *Service {
	return &Service{
		db:               db,
		eventsRepository: repo,
	}
}

func (s *Service) CreateEvent(ctx context.Context, info *model.EventCreate) (*model.Event, error) {
	repeatRule := ""
	if info.RepeatType != model.RepeatTypeNone {
		var err error
		repeatRule, err = getRule(info.RepeatType, info.From)
		if err != nil {
			return nil, err
		}
	}

	event := &model.Event{
		RepeatRule:  repeatRule,
		Exceptions:  map[time.Time]struct{}{},
		EventCreate: *info,
	}

	id, err := s.eventsRepository.CreateEvent(ctx, s.db, event)
	if err != nil {
		return nil, fmt.Errorf("eventsRepository.CreateEvent: %w", err)
	}

	event.ID = fmt.Sprintf("%v_%v", id, info.From.Unix())
	return event, nil
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

			if _, ok := e.Exceptions[r]; ok {
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

func getRule(t model.RepeatType, from time.Time) (string, error) {
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

	rule, err := rrule.NewRRule(rrule.ROption{
		Freq:     freq,
		Interval: interval,
		Dtstart:  from.UTC(),
	})
	if err != nil {
		return "", fmt.Errorf("creating rule: %w", err)
	}

	return rule.String(), nil
}

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

	if _, ok := event.Exceptions[ts]; ok {
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
