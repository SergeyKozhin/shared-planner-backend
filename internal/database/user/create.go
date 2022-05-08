package user

import (
	"context"
	"fmt"

	"github.com/SergeyKozhin/shared-planner-backend/internal/database"
	"github.com/SergeyKozhin/shared-planner-backend/internal/model"
)

func (*Repository) CreateUser(ctx context.Context, q database.Queryable, user *model.User) error {
	qb := database.PSQL.
		Insert(database.UsersTable).
		Columns("full_name", "email", "phone_number", "photo").
		Values(
			user.FullName,
			user.Email,
			user.PhoneNumber,
			user.Photo,
		).
		Suffix("returning id")

	var id int64
	if err := q.Get(ctx, &id, qb); err != nil {
		return fmt.Errorf("SQL request: %w", err)
	}

	user.ID = id
	return nil
}
