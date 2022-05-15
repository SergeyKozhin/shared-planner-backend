package api

import (
	"net/http"

	"github.com/SergeyKozhin/shared-planner-backend/internal/pkg/fcm"
)

func (a *Api) sendTestMessageHandler(w http.ResponseWriter, r *http.Request) {
	if err := a.fcm.SendMessage(r.Context(), &fcm.Message{
		Token: "cszU4KRuT3eEk7eRVIy_yN:APA91bHuxFCzIVQ1W928-guHpobB0DL5xEoOA4xsK-OILJMtXHIkOcTFptKs3lhBID07xUkZoM4SxRtAdrSZi1DSk5CwePWBCgSaqw6uAofdNoKzD-E7UUdVVPisARgP_eE5kNPuo4hV",
		Data: map[string]string{
			"event_type":        "0",
			"notification_type": "0",
			"event_title":       "some event",
			"group_id":          "4",
		},
	}); err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}
