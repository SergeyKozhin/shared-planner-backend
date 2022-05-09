package group

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/SergeyKozhin/shared-planner-backend/internal/database"
	"github.com/SergeyKozhin/shared-planner-backend/internal/model"
)

func (*Repository) UpdateGroupName(ctx context.Context, q database.Queryable, groupID int64, name string) error {
	qb := database.PSQL.
		Update(database.GroupsTable).
		Set("name", name).
		Where(sq.Eq{"id": groupID})

	if _, err := q.Exec(ctx, qb); err != nil {
		return fmt.Errorf("SQL request: %w", err)
	}

	return nil
}

func (*Repository) UpdateGroupSettings(ctx context.Context, q database.Queryable, settings *model.GroupSettings) error {
	qb := database.PSQL.
		Update(database.UserGroupTable).
		Set("color", "#"+settings.Color.ToHTML()).
		Set("notify", settings.Notify).
		Where(sq.Eq{"group_id": settings.GroupID, "user_id": settings.UserID})

	if _, err := q.Exec(ctx, qb); err != nil {
		return fmt.Errorf("SQL request: %w", err)
	}

	return nil
}

func (*Repository) AddUserToGroup(ctx context.Context, q database.Queryable, settings *model.GroupSettings) error {
	qb := database.PSQL.
		Insert(database.UserGroupTable).
		Columns("user_id", "group_id", "color", "notify").
		Values(
			settings.UserID,
			settings.GroupID,
			"#"+settings.Color.ToHTML(),
			settings.Notify,
		)

	if _, err := q.Exec(ctx, qb); err != nil {
		return fmt.Errorf("SQL request: %w", err)
	}

	return nil
}

func (*Repository) RemoveUserFromGroup(ctx context.Context, q database.Queryable, groupID int64, userID int64) error {
	qb := database.PSQL.
		Delete(database.UserGroupTable).
		Where(sq.Eq{"group_id": groupID, "user_id": userID})

	if _, err := q.Exec(ctx, qb); err != nil {
		return fmt.Errorf("SQL request: %w", err)
	}

	return nil
}
