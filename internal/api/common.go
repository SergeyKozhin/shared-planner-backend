package api

import (
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
