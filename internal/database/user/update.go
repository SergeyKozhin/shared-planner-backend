package user

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/SergeyKozhin/shared-planner-backend/internal/database"
)

func (*Repository) UpdateUserPushToken(ctx context.Context, q database.Queryable, id int64, token string) error {
	qb := database.PSQL.
		Update(database.UsersTable).
		Set("push_token", token).
		Where(sq.Eq{"id": id})

	if _, err := q.Exec(ctx, qb); err != nil {
		return fmt.Errorf("SQL request: %w", err)
	}

	return nil
}
