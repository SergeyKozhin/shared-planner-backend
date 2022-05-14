package events

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/SergeyKozhin/shared-planner-backend/internal/database"
	"github.com/SergeyKozhin/shared-planner-backend/internal/model"
)

func (*Repository) UpdateEvent(ctx context.Context, q database.Queryable, event *model.Event) error {
	notifications := make([]int64, len(event.Notifications))
	for i, n := range event.Notifications {
		notifications[i] = int64(n)
	}

	exceptions := make([]time.Time, 0, len(event.Exceptions))
	for e := range event.Exceptions {
		exceptions = append(exceptions, time.Unix(e, 0))
	}

	qb := database.PSQL.
		Update(database.EventsTable).
		SetMap(map[string]interface{}{
			"type":            event.EventType,
			"title":           event.Title,
			"description":     event.Description,
			"attachments":     event.Attachments,
			"notifications":   notifications,
			"group_id":        event.GroupID,
			"all_day":         event.AllDay,
			"repeat_type":     event.RepeatType,
			"start_date":      event.From,
			"end_date":        event.Until,
			"duration":        event.To.Sub(event.From),
			"recurrence_rule": event.RepeatRule,
			"exceptions":      exceptions,
		}).
		Where(sq.Eq{"id": event.ID})

	if _, err := q.Exec(ctx, qb); err != nil {
		return fmt.Errorf("SQL request: %w", err)
	}

	return nil
}
