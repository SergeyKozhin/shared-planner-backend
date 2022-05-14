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
	UpdateEvent(ctx context.Context, q database.Queryable, event *model.Event) error
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
