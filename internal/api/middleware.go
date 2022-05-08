package api

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/SergeyKozhin/shared-planner-backend/internal/model"
	"github.com/SergeyKozhin/shared-planner-backend/internal/pkg/jwt"
)

type contextKey string

const (
	contextKeyID   = contextKey("id")
	contextKeyUser = contextKey("user")
)

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
			a.serverErrorResponse(w, r, errors.New("can't retrieve id"))
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
