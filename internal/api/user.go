package api

import (
	"errors"
	"net/http"

	"github.com/SergeyKozhin/shared-planner-backend/internal/model"
)

var errCantRetrieveUser = errors.New("can't retrieve user from context")

func (a *Api) getUserHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(contextKeyUser).(*model.User)
	if !ok {
		a.serverErrorResponse(w, r, errCantRetrieveUser)
		return
	}

	resp := &struct {
		ID          int64  `json:"id,omitempty"`
		FullName    string `json:"full_name,omitempty"`
		Email       string `json:"email,omitempty"`
		PhoneNumber string `json:"phone_number,omitempty"`
		Photo       string `json:"photo,omitempty"`
	}{
		ID:          user.ID,
		FullName:    user.FullName,
		Email:       user.Email,
		PhoneNumber: user.PhoneNumber,
		Photo:       user.Photo,
	}

	if err := a.writeJSON(w, http.StatusOK, resp, nil); err != nil {
		a.serverErrorResponse(w, r, err)
	}
}
