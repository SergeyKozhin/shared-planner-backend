package group

import (
	"github.com/SergeyKozhin/shared-planner-backend/internal/database"
)

var baseQuery = database.PSQL.
	Select(
		"g.id",
		"g.name",
		"g.creator_id",
		"array_agg(ug.user_id) users_ids",
	).
	From(database.GroupsTable + " g").
	Join(database.UserGroupTable + " ug on g.id = ug.group_id").
	GroupBy("g.id")
