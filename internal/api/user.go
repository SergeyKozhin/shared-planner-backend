package api

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/SergeyKozhin/shared-planner-backend/internal/model"
	"github.com/SergeyKozhin/shared-planner-backend/internal/pkg/validator"
)

var errCantRetrieveUser = errors.New("can't retrieve user from context")

func (a *Api) getUserHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(contextKeyUser).(*model.User)
	if !ok {
		a.serverErrorResponse(w, r, errCantRetrieveUser)
		return
	}

	resp := &userResp{
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

func (a *Api) searchUsersHandler(w http.ResponseWriter, r *http.Request) {
	filter, err := parseQuery(r)
	if err != nil {
		a.badRequestResponse(w, r, err)
		return
	}

	users, err := a.users.SearchUsers(r.Context(), a.db, *filter)
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}

	usersResp, err := mapSlice(users, mapToUserResp)
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}

	resp := struct {
		Users    []*userResp `json:"users"`
		Page     int         `json:"page"`
		NextPage int         `json:"next_page"`
	}{
		Users:    usersResp,
		Page:     filter.Page,
		NextPage: filter.Page + 1,
	}

	if err := a.writeJSON(w, http.StatusOK, resp, nil); err != nil {
		a.serverErrorResponse(w, r, err)
	}
}

func parseQuery(r *http.Request) (*model.UserSearchFilter, error) {
	filter := &model.UserSearchFilter{
		Query: "",
		Limit: 20,
		Page:  1,
	}

	filter.Query = r.URL.Query().Get("query")
	if len(filter.Query) == 0 {
		return nil, errors.New("query must be provided")
	}

	if v := r.URL.Query().Get("limit"); v != "" {
		limit, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, errors.New("limit must be valid")
		}

		if limit < 1 {
			return nil, errors.New("limit must be valid")
		}

		filter.Limit = int(limit)
	}

	if v := r.URL.Query().Get("page"); v != "" {
		page, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, errors.New("page must be valid")
		}

		if page < 1 {
			return nil, errors.New("page must be valid")
		}

		filter.Page = int(page)
	}

	return filter, nil
}

func (a *Api) updateUserPushTokenHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(contextKeyUser).(*model.User)
	if !ok {
		a.serverErrorResponse(w, r, errCantRetrieveUser)
		return
	}

	req := &struct {
		PushToken string `json:"push_token"`
	}{}

	if err := a.readJSON(w, r, req); err != nil {
		a.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	v.Check(req.PushToken != "", "push_token", "push token must be provided")

	if !v.Valid() {
		a.failedValidationResponse(w, r, v.Errors)
		return
	}

	if err := a.users.UpdateUserPushToken(r.Context(), a.db, user.ID, req.PushToken); err != nil {
		a.serverErrorResponse(w, r, fmt.Errorf("update push token: %w", err))
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (a *Api) updateUserNotifyHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(contextKeyUser).(*model.User)
	if !ok {
		a.serverErrorResponse(w, r, errCantRetrieveUser)
		return
	}

	req := &struct {
		Notify bool `json:"notify"`
	}{}

	if err := a.readJSON(w, r, req); err != nil {
		a.badRequestResponse(w, r, err)
		return
	}

	if err := a.users.UpdateNotify(r.Context(), a.db, user.ID, req.Notify); err != nil {
		a.serverErrorResponse(w, r, fmt.Errorf("update notify: %w", err))
		return
	}

	w.WriteHeader(http.StatusOK)
}
