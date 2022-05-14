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
		Exceptions:  []time.Time{},
		EventCreate: *info,
	}

	id, err := s.eventsRepository.CreateEvent(ctx, s.db, event)
	if err != nil {
		return nil, fmt.Errorf("eventsRepository.CreateEvent: %w", err)
	}

	event.ID = id
	return event, nil
}

func (s *Service) GetEvents(ctx context.Context, filter model.EventsFilter) ([]*model.Event, error) {
	baseEvents, err := s.eventsRepository.GetEvents(ctx, s.db, filter)
	if err != nil {
		return nil, fmt.Errorf("eventsRepository.GetEvents: %w", err)
	}

	var res []*model.Event

	for _, e := range baseEvents {
		duration := e.To.Sub(e.From)

		if e.RepeatType == model.RepeatTypeNone {
			res = append(res, e)
			continue
		}

		rOption, err := rrule.StrToROption(e.RepeatRule)
		if err != nil {
			return nil, fmt.Errorf("parse repeat rule %q: %w", e.RepeatRule, err)
		}
		rule, err := rrule.NewRRule(*rOption)
		if err != nil {
			return nil, fmt.Errorf("make rule: %w", err)
		}

		exceptionsMap := make(map[time.Time]struct{}, len(e.Exceptions))
		for _, exc := range e.Exceptions {
			exceptionsMap[exc] = struct{}{}
		}

		repeats := rule.Between(filter.From, filter.To, true)
		for _, r := range repeats {
			if _, ok := exceptionsMap[r]; ok {
				continue
			}

			res = append(res, &model.Event{
				ID:         e.ID,
				RepeatRule: e.RepeatRule,
				Exceptions: e.Exceptions,
				EventCreate: model.EventCreate{
					GroupID:       e.GroupID,
					EventType:     e.EventType,
					Title:         e.Title,
					Description:   e.Description,
					AllDay:        e.AllDay,
					From:          r,
					To:            r.Add(duration),
					RepeatType:    e.RepeatType,
					Notifications: e.Notifications,
					Attachments:   e.Attachments,
				},
			})
		}
	}

	sort.Slice(res, func(i, j int) bool {
		if !res[i].From.Equal(res[j].From) {
			res[i].From.Before(res[j].From)
		}

		return res[i].ID < res[j].ID
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
