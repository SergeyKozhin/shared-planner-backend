package model

import "time"

type EventCreate struct {
	GroupID       int64
	EventType     EventType
	Title         string
	Description   string
	AllDay        bool
	From          time.Time
	To            time.Time
	RepeatType    RepeatType
	Notifications []time.Duration
	Attachments   []*Attachment
}
type Attachment struct {
	Name string
	Path string
}

type Event struct {
	ID         string
	RepeatRule string
	Exceptions map[time.Time]struct{}
	EventCreate
}

type EventType int

const (
	EventTypeEvent EventType = iota
	EventTypeNotification
)

type RepeatType int

const (
	RepeatTypeNone RepeatType = iota
	RepeatTypeEveryDay
	RepeatTypeEveryThreeDays
	RepeatTypeEveryWeek
	RepeatTypeEveryMonth
	RepeatTypeEveryYear
)

type EventsFilter struct {
	From     time.Time
	To       time.Time
	GroupIDs []int64
}
