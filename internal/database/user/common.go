package user

import (
	"github.com/SergeyKozhin/shared-planner-backend/internal/database"
)

var baseQuery = database.PSQL.
	Select(
		"id",
		"full_name",
		"email",
		"phone_number",
		"photo",
	).
	From(database.UsersTable)
