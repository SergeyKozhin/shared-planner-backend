package notifications

import (
	"context"
	"fmt"
	"time"

	"github.com/SergeyKozhin/shared-planner-backend/internal/database"
	"github.com/SergeyKozhin/shared-planner-backend/internal/model"
	"github.com/SergeyKozhin/shared-planner-backend/internal/pkg/fcm"
	"github.com/xlab/closer"
	"go.uber.org/zap"
)

type Sender struct {
	db            database.PGX
	logger        *zap.SugaredLogger
	groups        groupsRepository
	users         usersRepository
	eventsService eventsService
	fcm           fcmService
}

type groupsRepository interface {
	GetGroups(ctx context.Context, q database.Queryable, ids []int64) ([]*model.Group, error)
	GetUserGroupSettings(ctx context.Context, q database.Queryable, filter model.UserGroupSettingsFilter) ([]*model.GroupSettings, error)
}

type usersRepository interface {
	GetUsersByIDs(ctx context.Context, q database.Queryable, ids []int64) ([]*model.User, error)
}

type eventsService interface {
	GetEvents(ctx context.Context, filter model.EventsFilter) ([]*model.Event, error)
}

type fcmService interface {
	SendMessage(ctx context.Context, m *fcm.Message) error
	SendMessageBatch(ctx context.Context, ms []*fcm.Message) error
}

func NewSender(
	db database.PGX,
	logger *zap.SugaredLogger,
	groups groupsRepository,
	users usersRepository,
	eventsService eventsService,
	fcm fcmService,
) *Sender {
	return &Sender{
		db:            db,
		logger:        logger,
		groups:        groups,
		users:         users,
		eventsService: eventsService,
		fcm:           fcm,
	}
}

func (s *Sender) Start(ctx context.Context) {
	now := time.Now()

	from := now.Truncate(time.Minute)
	to := from.Add(time.Minute)
	// initial send
	go s.findAndSendNotifications(ctx, from, to)

	time.Sleep(to.Sub(time.Now()))

	// send at first minute
	from = to
	to = time.Now().Truncate(time.Minute).Add(time.Minute)
	go s.findAndSendNotifications(ctx, from, to)

	ticker := time.NewTicker(time.Minute)
	done := make(chan bool)

	closer.Bind(func() {
		done <- true
		ticker.Stop()
	})

	for {
		select {
		case <-done:
			break
		case t := <-ticker.C:
			from = to
			to = t.Truncate(time.Minute).Add(time.Minute)
			go s.findAndSendNotifications(ctx, from, to)
		}
	}
}

type notification struct {
	event  *model.Event
	notify time.Duration
}

func (s *Sender) findAndSendNotifications(ctx context.Context, from, to time.Time) {
	s.logger.Debugw("sending notifications", "from", from, "to", to)

	filter := model.EventsFilter{
		From:     from.Add(5 * time.Minute),
		To:       to.Add(24 * time.Hour),
		GroupIDs: nil,
	}
	events, err := s.eventsService.GetEvents(ctx, filter)
	if err != nil {
		s.logger.Errorw("failed to get events", "filter", filter, "err", err)
		return
	}

	notifications := getPossibleNotifications(events, from, to)

	groups, err := s.getGroups(ctx, notifications)
	if err != nil {
		s.logger.Errorw("failed to get groups")
		return
	}

	users, settings, err := s.getUsersAndSettings(ctx, groups)
	if err != nil {
		s.logger.Errorw("failed to get users and settings")
		return
	}

	if err := s.sendNotifications(ctx, notifications, groups, users, settings); err != nil {
		s.logger.Errorw("failed to send notifications: %w", err)
	}
}

func getPossibleNotifications(events []*model.Event, from, to time.Time) []*notification {
	var res []*notification
	for _, e := range events {
		for _, n := range e.Notifications {
			notifyTime := e.From.Add(-n)
			if !notifyTime.Before(from) && notifyTime.Before(to) {
				res = append(res, &notification{
					event:  e,
					notify: n,
				})
			}
		}
	}

	return res
}

func (s *Sender) getGroups(ctx context.Context, notification []*notification) (map[int64]*model.Group, error) {
	var groupIDs []int64
	groupIDsMap := make(map[int64]struct{})

	for _, n := range notification {
		if _, ok := groupIDsMap[n.event.GroupID]; !ok {
			groupIDs = append(groupIDs, n.event.GroupID)
			groupIDsMap[n.event.GroupID] = struct{}{}
		}
	}

	groups, err := s.groups.GetGroups(ctx, s.db, groupIDs)
	if err != nil {
		return nil, fmt.Errorf("get groups: %w", err)
	}

	res := make(map[int64]*model.Group, len(groups))
	for _, g := range groups {
		res[g.ID] = g
	}

	return res, nil
}

func (s *Sender) getUsersAndSettings(ctx context.Context, groups map[int64]*model.Group) (map[int64]*model.User, map[int64][]*model.GroupSettings, error) {
	var groupIDs []int64
	groupIDsMap := make(map[int64]struct{})

	var userIDs []int64
	userIDsMap := make(map[int64]struct{})

	for _, g := range groups {
		if _, ok := groupIDsMap[g.ID]; !ok {
			groupIDs = append(groupIDs, g.ID)
			groupIDsMap[g.ID] = struct{}{}
		}

		for _, id := range g.UsersIDs {
			if _, ok := userIDsMap[id]; !ok {
				userIDs = append(userIDs, id)
				userIDsMap[id] = struct{}{}
			}
		}
	}

	users, err := s.users.GetUsersByIDs(ctx, s.db, userIDs)
	if err != nil {
		return nil, nil, fmt.Errorf("get users: %w", err)
	}

	usersMap := make(map[int64]*model.User, len(users))
	for _, u := range users {
		usersMap[u.ID] = u
	}

	settings, err := s.groups.GetUserGroupSettings(ctx, s.db, model.UserGroupSettingsFilter{
		UserIDs:  userIDs,
		GroupIDs: groupIDs,
	})

	settingsMap := make(map[int64][]*model.GroupSettings)
	for _, s := range settings {
		settingsMap[s.UserID] = append(settingsMap[s.UserID], s)
	}

	return usersMap, settingsMap, nil
}

func (s *Sender) sendNotifications(
	ctx context.Context,
	notifications []*notification,
	groups map[int64]*model.Group,
	users map[int64]*model.User,
	settings map[int64][]*model.GroupSettings,
) error {
	var messages []*fcm.Message
	for _, n := range notifications {
		group, ok := groups[n.event.GroupID]
		if !ok {
			return fmt.Errorf("group not found %v", n.event.GroupID)
		}

		for _, userID := range group.UsersIDs {
			user, ok := users[userID]
			if !ok {
				return fmt.Errorf("user not found %v", userID)
			}
			if !user.Notify || user.PushToken == "" {
				continue
			}

			var groupSettings *model.GroupSettings
			for _, s := range settings[userID] {
				if s.GroupID == group.ID {
					groupSettings = s
				}
			}
			if groupSettings == nil {
				return fmt.Errorf("user group settings not found %v %v", userID, group.ID)
			}
			if !groupSettings.Notify {
				continue
			}

			notificationType, err := mapToDurationValue(n.notify)
			if err != nil {
				return fmt.Errorf("map notification type: %w", err)
			}
			messages = append(messages, &fcm.Message{
				Token: user.PushToken,
				Data: map[string]string{
					"event_type":        fmt.Sprintf("%v", n.event.EventType),
					"notification_type": fmt.Sprintf("%v", notificationType),
					"event_title":       n.event.Title,
					"group_id":          fmt.Sprintf("%v", n.event.GroupID),
				},
			})
		}
	}

	if err := s.fcm.SendMessageBatch(ctx, messages); err != nil {
		return fmt.Errorf("send notifications")
	}

	return nil
}
