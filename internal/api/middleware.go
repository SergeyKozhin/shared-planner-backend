package api

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/SergeyKozhin/shared-planner-backend/internal/pkg/jwt"
)

type contextKey string

const (
	contextKeyID = contextKey("id")
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
