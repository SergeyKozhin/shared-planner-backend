package model

import (
	"github.com/gerow/go-color"
)

type GroupCreate struct {
	Name      string
	CreatorID int64
}

type Group struct {
	ID       int64
	UsersIDs []int64
	GroupCreate
}

type GroupSettings struct {
	UserID  int64
	GroupID int64
	Color   color.RGB
	Notify  bool
}

type UserGroupSettingsFilter struct {
	UserIDs  []int64
	GroupIDs []int64
}
