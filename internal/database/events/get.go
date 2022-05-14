package events

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/SergeyKozhin/shared-planner-backend/internal/database"
	"github.com/SergeyKozhin/shared-planner-backend/internal/model"
)

func (*Repository) GetEventByID(ctx context.Context, q database.Queryable, id int64) (*model.Event, error) {
	qb := baseQuery.
		Where(sq.Eq{"id": id})

	var dto *eventDTO
	if err := q.Select(ctx, dto, qb); err != nil {
		return nil, fmt.Errorf("SQL request: %w", err)
	}

	return mapToEvent(dto), nil
}

func (*Repository) GetEvents(ctx context.Context, q database.Queryable, filter model.EventsFilter) ([]*model.Event, error) {
	qb := baseQuery.
		Where(sq.GtOrEq{"start_date": filter.From}).
		Where(sq.Or{sq.Eq{"end_date": nil}, sq.Lt{"end_date": filter.To}})

	if len(filter.GroupIDs) != 0 {
		qb = qb.Where(sq.Eq{"group_id": filter.GroupIDs})
	}

	var dtos []*eventDTO
	if err := q.Select(ctx, &dtos, qb); err != nil {
		return nil, fmt.Errorf("SQL request: %w", err)
	}

	res := make([]*model.Event, len(dtos))
	for i, d := range dtos {
		res[i] = mapToEvent(d)
	}

	return res, nil
}
