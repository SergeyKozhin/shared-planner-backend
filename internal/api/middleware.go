package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/SergeyKozhin/shared-planner-backend/internal/model"
	"github.com/SergeyKozhin/shared-planner-backend/internal/pkg/jwt"
	"github.com/go-chi/chi/v5"
)

type contextKey string

const (
	contextKeyID    = contextKey("id")
	contextKeyUser  = contextKey("user")
	contextKeyGroup = contextKey("group")
)

var errCantRetrieveID = errors.New("can't retrieve id")

func (a *Api) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			a.unauthorizedResponse(w, r, errors.New("no token provided"))
			return
		}

		token = strings.TrimPrefix(token, "Bearer ")

		id, err := a.jwts.GetIdFromToken(token)
		if err != nil {
			invalidTokenErr := &jwt.InvalidTokenError{}
			switch {
			case errors.As(err, &invalidTokenErr):
				a.unauthorizedResponse(w, r, invalidTokenErr)
			default:
				a.serverErrorResponse(w, r, err)
			}
			return
		}

		idContext := context.WithValue(r.Context(), contextKeyID, id)
		next.ServeHTTP(w, r.WithContext(idContext))
	})
}

func (a *Api) userCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := r.Context().Value(contextKeyID).(int64)
		if !ok {
			a.serverErrorResponse(w, r, errCantRetrieveID)
			return
		}

		user, err := a.users.GetUserByID(r.Context(), a.db, id)
		if err != nil {
			switch {
			case errors.Is(err, model.ErrNoRecord):
				a.forbiddenResponse(w, r, "user does not exists")
			default:
				a.serverErrorResponse(w, r, err)
			}
			return
		}

		userCtx := context.WithValue(r.Context(), contextKeyUser, user)
		next.ServeHTTP(w, r.WithContext(userCtx))
	})
}

func (a *Api) groupCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := r.Context().Value(contextKeyID).(int64)
		if !ok {
			a.serverErrorResponse(w, r, errCantRetrieveID)
			return
		}

		groupID, err := strconv.ParseInt(chi.URLParam(r, "groupID"), 10, 64)
		if err != nil {
			a.notFoundResponse(w, r)
			return
		}

		group, err := a.groups.GetGroup(r.Context(), a.db, groupID)
		if err != nil {
			switch {
			case errors.Is(err, model.ErrNoRecord):
				a.notFoundResponse(w, r)
			default:
				a.serverErrorResponse(w, r, fmt.Errorf("get group: %w", err))
			}
			return
		}

		userInGroup := false
		for _, id := range group.UsersIDs {
			if id == userID {
				userInGroup = true
				break
			}
		}

		if !userInGroup {
			a.notFoundResponse(w, r)
			return
		}

		groupCtx := context.WithValue(r.Context(), contextKeyGroup, group)
		next.ServeHTTP(w, r.WithContext(groupCtx))
	})
}
