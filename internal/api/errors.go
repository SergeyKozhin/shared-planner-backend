package api

import (
	"fmt"
	"net/http"
)

func (a *Api) logError(_ *http.Request, err error) {
	a.logger.Errorw("server error", "error", err)
}

func (a *Api) errorResponse(w http.ResponseWriter, r *http.Request, status int, message interface{}) {
	data := map[string]interface{}{"error": message}

	if err := a.writeJSON(w, status, data, nil); err != nil {
		a.logError(r, err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (a *Api) serverErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	a.logError(r, err)

	message := "the server encountered a problem and could not process your request"
	a.errorResponse(w, r, http.StatusInternalServerError, message)
}

func (a *Api) clientErrorResponse(w http.ResponseWriter, r *http.Request, status int, message interface{}) {
	a.logger.Debugw("client error", "err", message)
	a.errorResponse(w, r, status, message)
}

func (a *Api) notFoundResponse(w http.ResponseWriter, r *http.Request) {
	message := "the requested resource could not be found"
	a.clientErrorResponse(w, r, http.StatusNotFound, message)
}

func (a *Api) methodNotAllowedResponse(w http.ResponseWriter, r *http.Request) {
	message := fmt.Sprintf("the %s method is not supported for this resource", r.Method)
	a.clientErrorResponse(w, r, http.StatusMethodNotAllowed, message)
}

func (a *Api) badRequestResponse(w http.ResponseWriter, r *http.Request, err error) {
	a.clientErrorResponse(w, r, http.StatusBadRequest, err.Error())
}

func (a *Api) failedValidationResponse(w http.ResponseWriter, r *http.Request, errors map[string]string) {
	a.clientErrorResponse(w, r, http.StatusUnprocessableEntity, errors)
}

func (a *Api) unauthorizedResponse(w http.ResponseWriter, r *http.Request, err error) {
	a.clientErrorResponse(w, r, http.StatusUnauthorized, err.Error())
}

func (a *Api) forbiddenResponse(w http.ResponseWriter, r *http.Request, message string) {
	a.clientErrorResponse(w, r, http.StatusForbidden, message)
}
