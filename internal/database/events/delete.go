package events

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/SergeyKozhin/shared-planner-backend/internal/database"
)

func (*Repository) DeleteEvent(ctx context.Context, q database.Queryable, id int64) error {
	qb := database.PSQL.
		Delete(database.EventsTable).
		Where(sq.Eq{"id": id})

	if _, err := q.Exec(ctx, qb); err != nil {
		return fmt.Errorf("SQL request: %w", err)
	}

	return nil
}
