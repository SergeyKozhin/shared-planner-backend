package events

import (
	"strconv"
	"time"

	"github.com/SergeyKozhin/shared-planner-backend/internal/model"
)

type eventDTO struct {
	ID             int64
	EventType      int `db:"type"`
	Title          string
	Description    string
	Attachments    []*attachmentDTO
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

type attachmentDTO struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func mapToEvent(dto *eventDTO) *model.Event {
	notifications := make([]time.Duration, len(dto.Notifications))
	for i, n := range dto.Notifications {
		notifications[i] = time.Duration(n)
	}

	exceptions := make(map[int64]struct{}, len(dto.Exceptions))
	for _, e := range dto.Exceptions {
		exceptions[e.Unix()] = struct{}{}
	}

	attachments := make([]*model.Attachment, len(dto.Attachments))
	for i, a := range dto.Attachments {
		attachments[i] = &model.Attachment{
			Name: a.Name,
			Path: a.Path,
		}
	}

	return &model.Event{
		ID:         strconv.FormatInt(dto.ID, 10),
		RepeatRule: dto.RecurrenceRule,
		Exceptions: exceptions,
		Until:      dto.EndDate,
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
			Attachments:   attachments,
		},
	}
}
