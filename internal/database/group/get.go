package group

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/SergeyKozhin/shared-planner-backend/internal/database"
	"github.com/SergeyKozhin/shared-planner-backend/internal/model"
	"github.com/jackc/pgx/v4"
)

func (*Repository) GetGroup(ctx context.Context, q database.Queryable, id int64) (*model.Group, error) {
	qb := baseQuery.
		Where(sq.Eq{"g.id": id})

	dto := &groupDTO{}
	if err := q.Get(ctx, dto, qb); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNoRecord
		}
		return nil, fmt.Errorf("SQL request: %w", err)
	}

	return mapToGroup(dto), nil
}

func (*Repository) GetUserGroups(ctx context.Context, q database.Queryable, userID int64) ([]*model.Group, error) {
	qb := baseQuery.
		Join(database.UserGroupTable+" ug1 on g.id = ug1.group_id").
		Where(sq.Eq{"ug1.user_id": userID}).
		GroupBy("g.id", "ug1.id").
		OrderBy("ug1.id")

	var dtos []*groupDTO
	if err := q.Select(ctx, &dtos, qb); err != nil {
		return nil, fmt.Errorf("SQL request: %w", err)
	}

	res := make([]*model.Group, len(dtos))
	for i, d := range dtos {
		res[i] = mapToGroup(d)
	}

	return res, nil
}

func (*Repository) GetUserGroupSettings(ctx context.Context, q database.Queryable, filter model.UserGroupSettingsFilter) ([]*model.GroupSettings, error) {
	qb := database.PSQL.
		Select(
			"id",
			"user_id",
			"group_id",
			"color",
			"notify",
		).
		From(database.UserGroupTable).
		OrderBy("id")

	if len(filter.UserIDs) != 0 {
		qb = qb.Where(sq.Eq{"user_id": filter.UserIDs})
	}

	if len(filter.GroupIDs) != 0 {
		qb = qb.Where(sq.Eq{"group_id": filter.GroupIDs})
	}

	var dtos []*groupSettingsDTO
	if err := q.Select(ctx, &dtos, qb); err != nil {
		return nil, fmt.Errorf("SQL request: %w", err)
	}

	res := make([]*model.GroupSettings, len(dtos))
	for i, d := range dtos {
		var err error
		res[i], err = mapToGroupSettings(d)
		if err != nil {
			return nil, fmt.Errorf("map settings: %w", err)
		}
	}

	return res, nil
}
