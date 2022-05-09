package api

import (
	"errors"
	"net/http"

	"github.com/SergeyKozhin/shared-planner-backend/internal/config"
	"github.com/SergeyKozhin/shared-planner-backend/internal/model"
)

func (a *Api) signInGoogleHandler(w http.ResponseWriter, r *http.Request) {
	req := &struct {
		AuthCode string `json:"auth_code"`
	}{}

	if err := a.readJSON(w, r, req); err != nil {
		a.badRequestResponse(w, r, err)
	}

	tokenInfo, err := a.tokenParser.GetInfoGoogle(r.Context(), req.AuthCode)
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}

	user, err := a.users.GetUserByEmail(r.Context(), a.db, tokenInfo.Email)
	if err != nil {
		if errors.Is(err, model.ErrNoRecord) {
			photoName := ""
			if tokenInfo.Picture != "" {
				response, err := http.Get(tokenInfo.Picture)
				if err != nil {
					a.serverErrorResponse(w, r, err)
					return
				}
				defer response.Body.Close()

				photoName, err = savePhoto(response.Body)
				if err != nil {
					a.serverErrorResponse(w, r, err)
					return
				}
			}

			userCreate := &model.UserCreate{
				FullName:    tokenInfo.Name,
				Email:       tokenInfo.Email,
				Photo:       photoName,
				PhoneNumber: tokenInfo.PhoneNumber,
			}
			id, err := a.users.CreateUser(r.Context(), a.db, userCreate)
			if err != nil {
				a.serverErrorResponse(w, r, err)
				return
			}

			user = &model.User{ID: id, UserCreate: *userCreate}
		} else {
			a.serverErrorResponse(w, r, err)
			return
		}
	}

	tokens, err := a.generateTokens(r.Context(), user.ID)
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}

	if err := a.writeJSON(w, http.StatusOK, tokens, nil); err != nil {
		a.serverErrorResponse(w, r, err)
	}
}

func (a *Api) refreshTokenHandler(w http.ResponseWriter, r *http.Request) {
	input := &struct {
		RefreshToken string `json:"refresh_token"`
	}{}

	if err := a.readJSON(w, r, input); err != nil {
		a.badRequestResponse(w, r, err)
		return
	}

	id, err := a.refreshTokens.Get(r.Context(), input.RefreshToken)
	if err != nil {
		switch {
		case errors.Is(err, model.ErrNoRecord):
			a.unauthorizedResponse(w, r, errors.New("no such session"))
		default:
			a.serverErrorResponse(w, r, err)
		}
		return
	}

	accessToken, err := a.jwts.CreateToken(id)
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}

	newRefreshToken := ""
	for {
		newRefreshToken, err = a.generateRandomString(config.SessionTokenLength())
		if err != nil {
			a.serverErrorResponse(w, r, err)
			return
		}

		if err := a.refreshTokens.Refresh(r.Context(), input.RefreshToken, newRefreshToken); err != nil {
			if errors.Is(err, model.ErrAlreadyExists) {
				continue
			}
			a.serverErrorResponse(w, r, err)
			return
		}

		break
	}

	response := &struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
	}

	if err := a.writeJSON(w, http.StatusOK, response, nil); err != nil {
		a.serverErrorResponse(w, r, err)
	}
}

func (a *Api) logoutUserHandler(w http.ResponseWriter, r *http.Request) {
	input := &struct {
		RefreshToken string `json:"refresh_token"`
	}{}

	if err := a.readJSON(w, r, input); err != nil {
		a.badRequestResponse(w, r, err)
		return
	}

	if err := a.refreshTokens.Delete(r.Context(), input.RefreshToken); err != nil {
		switch {
		case errors.Is(err, model.ErrNoRecord):
			a.unauthorizedResponse(w, r, errors.New("no such session"))
		default:
			a.serverErrorResponse(w, r, err)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
}
