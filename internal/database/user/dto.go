package user

import (
	"github.com/SergeyKozhin/shared-planner-backend/internal/model"
)

type userDTO struct {
	ID          int64
	FullName    string
	Email       string
	PhoneNumber string
	Photo       string
	PushToken   string
	GroupsIDs   []int64
}

func mapToUser(dto *userDTO) *model.User {
	return &model.User{
		ID:        dto.ID,
		PushToken: dto.PushToken,
		UserCreate: model.UserCreate{
			FullName:    dto.FullName,
			Email:       dto.Email,
			PhoneNumber: dto.PhoneNumber,
			Photo:       dto.Photo,
		},
	}
}
