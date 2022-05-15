package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/SergeyKozhin/shared-planner-backend/internal/model"
	"github.com/SergeyKozhin/shared-planner-backend/internal/pkg/validator"
	"github.com/gerow/go-color"
)

var errCantRetrieveGroup = errors.New("can't retrieve group from context")

func (a *Api) getUserGroupsHandler(w http.ResponseWriter, r *http.Request) {
	type getUserGroupsResponse struct {
		ID        int64  `json:"id"`
		Name      string `json:"name"`
		Color     string `json:"color"`
		Notify    bool   `json:"notify"`
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
			ID:        g.ID,
			Name:      g.Name,
			Color:     "#" + s.Color.ToHTML(),
			Notify:    s.Notify,
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

	toAdd := []int64{userID}
	toAddMap := make(map[int64]struct{})
	toAddMap[userID] = struct{}{}

	for _, id := range req.UsersIDs {
		if _, ok := toAddMap[id]; !ok {
			toAdd = append(toAdd, id)
			toAddMap[id] = struct{}{}
		}
	}

	for _, user := range toAdd {
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

func (a *Api) getGroupHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(contextKeyID).(int64)
	if !ok {
		a.serverErrorResponse(w, r, errCantRetrieveID)
		return
	}

	group, ok := r.Context().Value(contextKeyGroup).(*model.Group)
	if !ok {
		a.serverErrorResponse(w, r, errCantRetrieveGroup)
		return
	}

	users, err := a.users.GetUsersByIDs(r.Context(), a.db, group.UsersIDs)
	if err != nil {
		a.serverErrorResponse(w, r, fmt.Errorf("get users: %w", err))
		return
	}

	settings, err := a.groups.GetUserGroupSettings(r.Context(), a.db, model.UserGroupSettingsFilter{
		UserIDs:  []int64{userID},
		GroupIDs: []int64{group.ID},
	})
	if err != nil {
		a.serverErrorResponse(w, r, fmt.Errorf("get group settings: %w", err))
		return
	}
	if len(settings) != 1 {
		a.serverErrorResponse(w, r, fmt.Errorf("invalid number of group settings %d", len(settings)))
		return
	}

	userResps, err := mapSlice(users, mapToUserResp)
	if err != nil {
		a.serverErrorResponse(w, r, fmt.Errorf("map users: %w", err))
		return
	}

	resp := &struct {
		ID        int64       `json:"id"`
		Name      string      `json:"name"`
		CreatorID int64       `json:"creator_id"`
		Color     string      `json:"color"`
		Users     []*userResp `json:"users"`
	}{
		ID:        group.ID,
		Name:      group.Name,
		CreatorID: group.CreatorID,
		Color:     "#" + settings[0].Color.ToHTML(),
		Users:     userResps,
	}

	if err := a.writeJSON(w, http.StatusOK, resp, nil); err != nil {
		a.serverErrorResponse(w, r, err)
	}
}

func (a *Api) updateGroupHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(contextKeyID).(int64)
	if !ok {
		a.serverErrorResponse(w, r, errCantRetrieveID)
		return
	}

	group, ok := r.Context().Value(contextKeyGroup).(*model.Group)
	if !ok {
		a.serverErrorResponse(w, r, errCantRetrieveGroup)
		return
	}

	req := &struct {
		Name     string  `json:"name"`
		UsersIDs []int64 `json:"users_ids"`
	}{}

	if err := a.readJSON(w, r, req); err != nil {
		a.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	v.Check(len(req.Name) != 0, "name", "name must be provided")
	v.Check(group.Name == req.Name || group.CreatorID == userID, "name", "only creator can change name")

	toAdd, toRemove, err := calculateUsers(group, req.UsersIDs, userID)
	if err != nil {
		v.AddError("users_ids", err.Error())
	}

	if !v.Valid() {
		a.failedValidationResponse(w, r, v.Errors)
		return
	}

	settings, err := a.groups.GetUserGroupSettings(r.Context(), a.db, model.UserGroupSettingsFilter{
		UserIDs:  []int64{group.CreatorID},
		GroupIDs: []int64{group.ID},
	})
	if err != nil {
		a.serverErrorResponse(w, r, fmt.Errorf("get group settings: %w", err))
		return
	}
	if len(settings) != 1 {
		a.serverErrorResponse(w, r, fmt.Errorf("invalid number of group settings %d", len(settings)))
		return
	}

	tx, err := a.db.BeginTx(r.Context(), nil)
	if err != nil {
		a.serverErrorResponse(w, r, fmt.Errorf("begin tx: %w", err))
		return
	}
	defer tx.Rollback(r.Context())

	if err := a.groups.UpdateGroupName(r.Context(), tx, group.ID, req.Name); err != nil {
		a.serverErrorResponse(w, r, fmt.Errorf("update group name: %w", err))
		return
	}

	for _, id := range toAdd {
		if err := a.groups.AddUserToGroup(r.Context(), tx, &model.GroupSettings{
			UserID:  id,
			GroupID: group.ID,
			Color:   settings[0].Color,
			Notify:  true,
		}); err != nil {
			a.serverErrorResponse(w, r, fmt.Errorf("add user to group: %w", err))
			return
		}
	}

	for _, id := range toRemove {
		if err := a.groups.RemoveUserFromGroup(r.Context(), tx, group.ID, id); err != nil {
			a.serverErrorResponse(w, r, fmt.Errorf("remove user from group: %w", err))
			return
		}
	}

	if err := tx.Commit(r.Context()); err != nil {
		a.serverErrorResponse(w, r, fmt.Errorf("commit tx: %w", err))
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (a *Api) updateGroupSettingsHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(contextKeyID).(int64)
	if !ok {
		a.serverErrorResponse(w, r, errCantRetrieveID)
		return
	}

	group, ok := r.Context().Value(contextKeyGroup).(*model.Group)
	if !ok {
		a.serverErrorResponse(w, r, errCantRetrieveGroup)
		return
	}

	req := &struct {
		Color  string `json:"color"`
		Notify bool   `json:"notify"`
	}{}

	if err := a.readJSON(w, r, req); err != nil {
		a.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	v.Check(validator.Matches(req.Color, validator.HexRX), "color", "color must be valid HEX color")

	if !v.Valid() {
		a.failedValidationResponse(w, r, v.Errors)
		return
	}

	colorRGB, err := color.HTMLToRGB(req.Color)
	if err != nil {
		a.serverErrorResponse(w, r, fmt.Errorf("parse color: %w", err))
		return
	}

	if err := a.groups.UpdateGroupSettings(r.Context(), a.db, &model.GroupSettings{
		UserID:  userID,
		GroupID: group.ID,
		Color:   colorRGB,
		Notify:  req.Notify,
	}); err != nil {
		a.serverErrorResponse(w, r, fmt.Errorf("update group settings: %w", err))
		return
	}

	w.WriteHeader(http.StatusOK)
}

func calculateUsers(group *model.Group, newUsers []int64, userID int64) ([]int64, []int64, error) {
	oldMap := make(map[int64]struct{})
	for _, id := range group.UsersIDs {
		oldMap[id] = struct{}{}
	}

	newMap := make(map[int64]struct{})
	for _, id := range newUsers {
		newMap[id] = struct{}{}
	}

	var toAdd []int64
	toAddMap := make(map[int64]struct{})

	var toRemove []int64
	toRemoveMap := make(map[int64]struct{})

	for _, id := range group.UsersIDs {
		if _, ok := newMap[id]; !ok {
			if _, ok := toRemoveMap[id]; !ok {
				toRemove = append(toRemove, id)
				toRemoveMap[id] = struct{}{}
			}
		}
	}

	for _, id := range newUsers {
		if _, ok := oldMap[id]; !ok {
			if _, ok := toAddMap[id]; !ok {
				toAdd = append(toAdd, id)
				toAddMap[id] = struct{}{}
			}
		}
	}

	if _, ok := toRemoveMap[group.CreatorID]; ok {
		return nil, nil, fmt.Errorf("can't remove creator")
	}

	if len(toRemove) != 0 && group.CreatorID != userID {
		return nil, nil, fmt.Errorf("only creator can remove users from group")
	}

	return toAdd, toRemove, nil
}
