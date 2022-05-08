package user

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/SergeyKozhin/shared-planner-backend/internal/database"
	"github.com/SergeyKozhin/shared-planner-backend/internal/model"
)

func (*Repository) GetUserByEmail(ctx context.Context, q database.Queryable, email string) (*model.User, error) {
	users, err := getUsers(ctx, q, sq.Eq{"email": email})
	if err != nil {
		return nil, err
	}

	if len(users) == 0 {
		return nil, model.ErrNoRecord
	}

	return users[0], nil
}

func getUsers(ctx context.Context, q database.Queryable, predicate interface{}) ([]*model.User, error) {
	qb := database.PSQL.
		Select(
			"id",
			"full_name",
			"email",
			"phone_number",
			"photo",
		).
		From(database.UsersTable).
		Where(predicate)

	var dtos []*userDTO
	if err := q.Select(ctx, &dtos, qb); err != nil {
		return nil, fmt.Errorf("SQL request: %w", err)
	}

	res := make([]*model.User, len(dtos))
	for i, d := range dtos {
		res[i] = mapToUser(d)
	}

	return res, nil
}
