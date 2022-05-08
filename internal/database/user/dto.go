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
}

func mapToUser(dto *userDTO) *model.User {
	return &model.User{
		ID:          dto.ID,
		FullName:    dto.FullName,
		Email:       dto.Email,
		PhoneNumber: dto.PhoneNumber,
		Photo:       dto.Photo,
	}
}
