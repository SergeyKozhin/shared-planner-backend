package api

import (
	"fmt"
	"net/http"

	"github.com/SergeyKozhin/shared-planner-backend/internal/model"
	"github.com/SergeyKozhin/shared-planner-backend/internal/pkg/validator"
	"github.com/gerow/go-color"
)

func (a *Api) getUserGroupsHandler(w http.ResponseWriter, r *http.Request) {
	type getUserGroupsResponse struct {
		GroupID   int64  `json:"group_id"`
		Name      string `json:"name"`
		Color     string `json:"color"`
		UserCount int    `json:"user_count"`
	}

	userID, ok := r.Context().Value(contextKeyID).(int64)
	if !ok {
		a.serverErrorResponse(w, r, errCantRetrieveID)
		return
	}

	groups, err := a.groups.GetUserGroups(r.Context(), a.db, userID)
	if err != nil {
		a.serverErrorResponse(w, r, fmt.Errorf("get groups by user id %v: %w", userID, err))
		return
	}

	settings, err := a.groups.GetUserGroupSettings(r.Context(), a.db, model.UserGroupSettingsFilter{UserIDs: []int64{userID}})
	if err != nil {
		a.serverErrorResponse(w, r, fmt.Errorf("get groups settings for user %v: %w", userID, err))
		return
	}

	settingsMap := make(map[int64]*model.GroupSettings)
	for _, s := range settings {
		settingsMap[s.GroupID] = s
	}

	resp := make([]getUserGroupsResponse, len(groups))
	for i, g := range groups {
		s, ok := settingsMap[g.ID]
		if !ok {
			a.serverErrorResponse(w, r, fmt.Errorf("no settings for group %d", g.ID))
			return
		}

		resp[i] = getUserGroupsResponse{
			GroupID:   g.ID,
			Name:      g.Name,
			Color:     "#" + s.Color.ToHTML(),
			UserCount: len(g.UsersIDs),
		}
	}

	if err := a.writeJSON(w, http.StatusOK, resp, nil); err != nil {
		a.serverErrorResponse(w, r, err)
	}
}

func (a *Api) createGroupHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(contextKeyID).(int64)
	if !ok {
		a.serverErrorResponse(w, r, errCantRetrieveID)
		return
	}

	req := &struct {
		Name     string  `json:"name"`
		UsersIDs []int64 `json:"users_ids"`
		Color    string  `json:"color"`
	}{}

	if err := a.readJSON(w, r, req); err != nil {
		a.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()

	v.Check(len(req.Name) != 0, "name", "name must be provided")
	v.Check(validator.Matches(req.Color, validator.HexRX), "color", "color must be valid HEX color")

	if !v.Valid() {
		a.failedValidationResponse(w, r, v.Errors)
		return
	}

	tx, err := a.db.BeginTx(r.Context(), nil)
	if err != nil {
		a.serverErrorResponse(w, r, fmt.Errorf("tx begin: %w", err))
		return
	}
	defer tx.Rollback(r.Context())

	colorRGB, err := color.HTMLToRGB(req.Color)
	if err != nil {
		a.serverErrorResponse(w, r, fmt.Errorf("parse color: %w", err))
		return
	}
	groupID, err := a.groups.CreateGroup(r.Context(), tx, &model.GroupCreate{
		Name:      req.Name,
		CreatorID: userID,
	})
	if err != nil {
		a.serverErrorResponse(w, r, fmt.Errorf("create group: %w", err))
		return
	}

	usersToAdd := append([]int64{userID}, req.UsersIDs...)
	for _, user := range usersToAdd {
		if err := a.groups.AddUserToGroup(r.Context(), tx, &model.GroupSettings{
			UserID:  user,
			GroupID: groupID,
			Color:   colorRGB,
			Notify:  true,
		}); err != nil {
			a.serverErrorResponse(w, r, fmt.Errorf("add user to group: %w", err))
			return
		}
	}

	if err := tx.Commit(r.Context()); err != nil {
		a.serverErrorResponse(w, r, fmt.Errorf("commit tx: %w", err))
	}

	w.WriteHeader(http.StatusCreated)
}
