package api

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	"github.com/SergeyKozhin/shared-planner-backend/internal/model"
)

type userResp struct {
	ID          int64  `json:"id,omitempty"`
	FullName    string `json:"full_name,omitempty"`
	Email       string `json:"email,omitempty"`
	PhoneNumber string `json:"phone_number,omitempty"`
	Photo       string `json:"photo,omitempty"`
}

func mapToUserResp(user *model.User) (*userResp, error) {
	return &userResp{
		ID:          user.ID,
		FullName:    user.FullName,
		Email:       user.Email,
		PhoneNumber: user.PhoneNumber,
		Photo:       user.Photo,
	}, nil
}

type attachment struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type eventResp struct {
	ID            string           `json:"id"`
	GroupID       int64            `json:"group_id"`
	EventType     model.EventType  `json:"event_type"`
	Title         string           `json:"title"`
	Description   string           `json:"description"`
	AllDay        bool             `json:"all_day"`
	From          dateTime         `json:"from"`
	To            dateTime         `json:"to"`
	RepeatType    model.RepeatType `json:"repeat_type"`
	Notifications []duration       `json:"notifications"`
	Attachments   []*attachment    `json:"attachments"`
}

func mapToEventsResp(event *model.Event) (*eventResp, error) {
	notifications, _ := mapSlice(event.Notifications, func(d time.Duration) (duration, error) {
		return duration(d), nil
	})

	attachments := make([]*attachment, len(event.Attachments))
	for i, a := range event.Attachments {
		attachments[i] = &attachment{
			Name: a.Name,
			Path: a.Path,
		}
	}

	return &eventResp{
		ID:            event.ID,
		GroupID:       event.GroupID,
		EventType:     event.EventType,
		Title:         event.Title,
		Description:   event.Description,
		AllDay:        event.AllDay,
		From:          dateTime(event.From),
		To:            dateTime(event.To),
		RepeatType:    event.RepeatType,
		Notifications: notifications,
		Attachments:   attachments,
	}, nil
}

type dateTime time.Time

var dateTimeFormat = "2006-01-02T15:04:05-07:00"

func (d dateTime) MarshalJSON() ([]byte, error) {
	res := []byte(fmt.Sprintf("%q", time.Time(d).Format(dateTimeFormat)))
	return res, nil
}

func (d *dateTime) UnmarshalJSON(b []byte) error {
	ts, err := time.Parse(dateTimeFormat, string(bytes.Trim(b, "\"")))
	if err != nil {
		return err
	}

	*d = dateTime(ts)
	return nil
}

type durationValue int

const (
	durationValue5Minutes durationValue = iota
	durationValue10Minutes
	durationValue15Minutes
	durationValue30Minutes
	durationValueHour
	durationValueDay
)

type duration time.Duration

func (d duration) MarshalJSON() ([]byte, error) {
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
		return nil, fmt.Errorf("unsupported duration: %v", d)
	}

	return []byte(fmt.Sprintf("%d", val)), nil
}

func (d *duration) UnmarshalJSON(b []byte) error {
	val, err := strconv.Atoi(string(b))
	if err != nil {
		return err
	}

	switch durationValue(val) {
	case durationValue5Minutes:
		*d = duration(5 * time.Minute)
	case durationValue10Minutes:
		*d = duration(10 * time.Minute)
	case durationValue15Minutes:
		*d = duration(15 * time.Minute)
	case durationValue30Minutes:
		*d = duration(30 * time.Minute)
	case durationValueHour:
		*d = duration(1 * time.Hour)
	case durationValueDay:
		*d = duration(24 * time.Hour)
	default:
		return fmt.Errorf("unknow duration value: %v", val)
	}

	return nil
}
