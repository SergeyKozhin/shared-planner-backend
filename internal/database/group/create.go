package group

import (
	"context"
	"fmt"

	"github.com/SergeyKozhin/shared-planner-backend/internal/database"
	"github.com/SergeyKozhin/shared-planner-backend/internal/model"
)

func (*Repository) CreateGroup(ctx context.Context, q database.Queryable, group *model.GroupCreate) (int64, error) {
	qb := database.PSQL.
		Insert(database.GroupsTable).
		Columns("name", "creator_id").
		Values(group.Name, group.CreatorID).
		Suffix("returning id")

	var id int64
	if err := q.Get(ctx, &id, qb); err != nil {
		return 0, fmt.Errorf("SQL request: %w", err)
	}

	return id, nil
}
