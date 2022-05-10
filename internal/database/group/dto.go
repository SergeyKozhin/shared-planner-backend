package group

import (
	"fmt"

	"github.com/SergeyKozhin/shared-planner-backend/internal/model"
	"github.com/gerow/go-color"
)

type groupDTO struct {
	ID        int64
	Name      string
	CreatorID int64
	UsersIDs  []int64 `db:"users_ids"`
}

func mapToGroup(d *groupDTO) *model.Group {
	return &model.Group{
		ID:       d.ID,
		UsersIDs: d.UsersIDs,
		GroupCreate: model.GroupCreate{
			Name:      d.Name,
			CreatorID: d.CreatorID,
		},
	}
}

type groupSettingsDTO struct {
	ID      int64
	UserID  int64
	GroupID int64
	Color   string
	Notify  bool
}

func mapToGroupSettings(d *groupSettingsDTO) (*model.GroupSettings, error) {
	colorRGB, err := color.HTMLToRGB(d.Color)
	if err != nil {
		return nil, fmt.Errorf("map color from %v", d.Color)
	}

	return &model.GroupSettings{
		UserID:  d.UserID,
		GroupID: d.GroupID,
		Color:   colorRGB,
		Notify:  d.Notify,
	}, nil
}
