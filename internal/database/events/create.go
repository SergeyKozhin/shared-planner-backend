package events

import (
	"context"
	"fmt"
	"time"

	"github.com/SergeyKozhin/shared-planner-backend/internal/database"
	"github.com/SergeyKozhin/shared-planner-backend/internal/model"
)

func (*Repository) CreateEvent(ctx context.Context, q database.Queryable, event *model.Event) (int64, error) {
	notifications := make([]int64, len(event.Notifications))
	for i, n := range event.Notifications {
		notifications[i] = int64(n)
	}

	var endDate *time.Time
	if event.RepeatType == model.RepeatTypeNone {
		endDate = &event.To
	}

	qb := database.PSQL.
		Insert(database.EventsTable).
		Columns(
			"type",
			"title",
			"description",
			"attachments",
			"notifications",
			"group_id",
			"all_day",
			"repeat_type",
			"start_date",
			"end_date",
			"duration",
			"recurrence_rule",
		).
		Values(
			event.EventType,
			event.Title,
			event.Description,
			event.Attachments,
			event.Notifications,
			event.GroupID,
			event.AllDay,
			event.RepeatType,
			event.From,
			endDate,
			event.To.Sub(event.From),
			event.RepeatRule,
		).
		Suffix("returning id")

	var id int64
	if err := q.Get(ctx, &id, qb); err != nil {
		return 0, fmt.Errorf("SQL request: %w", err)
	}

	return id, nil
}
