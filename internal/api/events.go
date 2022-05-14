package api

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/SergeyKozhin/shared-planner-backend/internal/model"
	"github.com/SergeyKozhin/shared-planner-backend/internal/pkg/validator"
)

var errCantRetrieveUserGroups = errors.New("can't retrieve user groups from context")
var errCantRetrieveEvent = errors.New("can't retrieve event from context")

func (a *Api) createEventHandler(w http.ResponseWriter, r *http.Request) {
	userGroups, ok := r.Context().Value(contextKeyUserGroups).(map[int64]struct{})
	if !ok {
		a.serverErrorResponse(w, r, errCantRetrieveUserGroups)
		return
	}

	req := &struct {
		GroupID       int64            `json:"group_id"`
		EventType     model.EventType  `json:"event_type"`
		Title         string           `json:"title"`
		Description   string           `json:"description"`
		AllDay        bool             `json:"all_day"`
		From          dateTime         `json:"from"`
		To            dateTime         `json:"to"`
		RepeatType    model.RepeatType `json:"repeat_type"`
		Notifications []duration       `json:"notifications"`
		Attachments   []*attachment    `json:"attachments"`
	}{}

	if err := a.readJSON(w, r, req); err != nil {
		a.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()

	_, ok = userGroups[req.GroupID]
	v.Check(ok, "group_id", "user does not have access to group")
	v.Check(len(req.Title) != 0, "title", "title must be provided")
	v.Check(!time.Time(req.From).IsZero(), "from", "from must be provided")

	if req.EventType == model.EventTypeEvent {
		v.Check(!time.Time(req.To).IsZero(), "to", "to must be provided")
	}

	if !v.Valid() {
		a.failedValidationResponse(w, r, v.Errors)
		return
	}

	notifications, _ := mapSlice(req.Notifications, func(d duration) (time.Duration, error) {
		return time.Duration(d), nil
	})

	attachments, _ := mapSlice(req.Attachments, func(a *attachment) (*model.Attachment, error) {
		return &model.Attachment{
			Name: a.Name,
			Path: a.Path,
		}, nil
	})

	if _, err := a.eventsService.CreateEvent(r.Context(), &model.EventCreate{
		GroupID:       req.GroupID,
		EventType:     req.EventType,
		Title:         req.Title,
		Description:   req.Description,
		AllDay:        req.AllDay,
		From:          time.Time(req.From),
		To:            time.Time(req.To),
		RepeatType:    req.RepeatType,
		Notifications: notifications,
		Attachments:   attachments,
	}); err != nil {
		a.serverErrorResponse(w, r, fmt.Errorf("create event: %w", err))
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (a *Api) getEventsHandler(w http.ResponseWriter, r *http.Request) {
	userGroups, ok := r.Context().Value(contextKeyUserGroups).(map[int64]struct{})
	if !ok {
		a.serverErrorResponse(w, r, errCantRetrieveUserGroups)
		return
	}

	filter, err := parseEventsQuery(r)
	if err != nil {
		a.badRequestResponse(w, r, err)
		return
	}

	for _, g := range filter.GroupIDs {
		if _, ok := userGroups[g]; !ok {
			a.forbiddenResponse(w, r, fmt.Sprintf("no acces for group %v", g))
			return
		}
	}

	events, err := a.eventsService.GetEvents(r.Context(), *filter)
	if err != nil {
		a.serverErrorResponse(w, r, fmt.Errorf("get events: %w", err))
		return
	}

	resp, _ := mapSlice(events, mapToEventsResp)

	if err := a.writeJSON(w, http.StatusOK, resp, nil); err != nil {
		a.serverErrorResponse(w, r, err)
	}
}

func parseEventsQuery(r *http.Request) (*model.EventsFilter, error) {
	var err error

	res := &model.EventsFilter{}

	v := r.URL.Query().Get("from")
	if v == "" {
		return nil, fmt.Errorf("from must be provided")
	}
	res.From, err = time.Parse(dateTimeFormat, v)
	if err != nil {
		return nil, fmt.Errorf("invalid time format: %w", err)
	}

	v = r.URL.Query().Get("to")
	if v == "" {
		return nil, fmt.Errorf("to must be provided")
	}
	res.To, err = time.Parse(dateTimeFormat, v)
	if err != nil {
		return nil, fmt.Errorf("invalid time format: %w", err)
	}

	vals := r.URL.Query()["group_ids"]
	res.GroupIDs = make([]int64, len(vals))
	for i, v := range vals {
		res.GroupIDs[i], err = strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid group id %v", v)
		}
	}

	return res, nil
}

func (a *Api) getEventHandler(w http.ResponseWriter, r *http.Request) {
	event, ok := r.Context().Value(contextKeyEvent).(*model.Event)
	if !ok {
		a.serverErrorResponse(w, r, errCantRetrieveEvent)
		return
	}

	resp, _ := mapToEventsResp(event)
	if err := a.writeJSON(w, http.StatusOK, resp, nil); err != nil {
		a.serverErrorResponse(w, r, err)
	}
}

func (a *Api) updateEventHandler(w http.ResponseWriter, r *http.Request) {
	event, ok := r.Context().Value(contextKeyEvent).(*model.Event)
	if !ok {
		a.serverErrorResponse(w, r, errCantRetrieveEvent)
		return
	}

	userGroups, ok := r.Context().Value(contextKeyUserGroups).(map[int64]struct{})
	if !ok {
		a.serverErrorResponse(w, r, errCantRetrieveUserGroups)
		return
	}

	req := &struct {
		OnlyUpdateInstance bool            `json:"only_update_instance"`
		GroupID            int64           `json:"group_id"`
		EventType          model.EventType `json:"event_type"`
		Title              string          `json:"title"`
		Description        string          `json:"description"`
		AllDay             bool            `json:"all_day"`
		From               dateTime        `json:"from"`
		To                 dateTime        `json:"to"`
		Notifications      []duration      `json:"notifications"`
	}{}

	if err := a.readJSON(w, r, req); err != nil {
		a.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()

	_, ok = userGroups[req.GroupID]
	v.Check(ok, "group_id", "user does not have access to group")
	v.Check(len(req.Title) != 0, "title", "title must be provided")
	v.Check(!time.Time(req.From).IsZero(), "from", "from must be provided")

	if req.EventType == model.EventTypeEvent {
		v.Check(!time.Time(req.To).IsZero(), "to", "to must be provided")
	}

	if !v.Valid() {
		a.failedValidationResponse(w, r, v.Errors)
		return
	}

	notifications, _ := mapSlice(req.Notifications, func(d duration) (time.Duration, error) {
		return time.Duration(d), nil
	})

	id, ts, err := splitID(event.ID)
	if err != nil {
		a.serverErrorResponse(w, r, fmt.Errorf("split id: %w", event))
		return
	}

	updateEvent := &model.EventUpdate{
		GroupID:       req.GroupID,
		EventType:     req.EventType,
		Title:         req.Title,
		Description:   req.Description,
		AllDay:        req.AllDay,
		From:          time.Time(req.From),
		To:            time.Time(req.To),
		Notifications: notifications,
	}

	if event.RepeatType == model.RepeatTypeNone || !req.OnlyUpdateInstance {
		if err := a.eventsService.UpdateEvent(r.Context(), id, ts, updateEvent); err != nil {
			a.serverErrorResponse(w, r, fmt.Errorf("update event: %w", err))
			return
		}
	} else {
		if err := a.eventsService.UpdateEventInstance(r.Context(), id, ts, updateEvent); err != nil {
			a.serverErrorResponse(w, r, fmt.Errorf("update event instance: %w", err))
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

func (a *Api) deleteEventHandler(w http.ResponseWriter, r *http.Request) {
	event, ok := r.Context().Value(contextKeyEvent).(*model.Event)
	if !ok {
		a.serverErrorResponse(w, r, errCantRetrieveEvent)
		return
	}

	req := &struct {
		OnlyDeleteInstance bool `json:"only_delete_instance"`
	}{}

	if err := a.readJSON(w, r, req); err != nil {
		a.badRequestResponse(w, r, err)
		return
	}

	id, ts, err := splitID(event.ID)
	if err != nil {
		a.serverErrorResponse(w, r, fmt.Errorf("split id: %w", event))
		return
	}

	if event.RepeatType == model.RepeatTypeNone || !req.OnlyDeleteInstance {
		if err := a.eventsService.DeleteEvent(r.Context(), id); err != nil {
			a.serverErrorResponse(w, r, fmt.Errorf("delete event: %w", err))
			return
		}
	} else {
		if err := a.eventsService.DeleteEventInstance(r.Context(), id, ts); err != nil {
			a.serverErrorResponse(w, r, fmt.Errorf("update event instance: %w", err))
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}
