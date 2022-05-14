package events

import (
	"time"

	"github.com/SergeyKozhin/shared-planner-backend/internal/model"
)

type eventDTO struct {
	ID             int64
	EventType      int `db:"type"`
	Title          string
	Description    string
	Attachments    []string
	Notifications  []int64
	GroupID        int64
	AllDay         bool
	RepeatType     int
	StartDate      time.Time
	EndDate        *time.Time
	Duration       time.Duration
	RecurrenceRule string
	Exceptions     []time.Time
}

func mapToEvent(dto *eventDTO) *model.Event {
	notifications := make([]time.Duration, len(dto.Notifications))
	for i, n := range dto.Notifications {
		notifications[i] = time.Duration(n)
	}

	return &model.Event{
		ID:         dto.ID,
		RepeatRule: dto.RecurrenceRule,
		Exceptions: dto.Exceptions,
		EventCreate: model.EventCreate{
			GroupID:       dto.GroupID,
			EventType:     model.EventType(dto.EventType),
			Title:         dto.Title,
			Description:   dto.Description,
			AllDay:        dto.AllDay,
			From:          dto.StartDate,
			To:            dto.StartDate.Add(dto.Duration),
			RepeatType:    model.RepeatType(dto.RepeatType),
			Notifications: notifications,
			Attachments:   dto.Attachments,
		},
	}
}
