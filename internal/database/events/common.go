package events

import "github.com/SergeyKozhin/shared-planner-backend/internal/database"

var baseQuery = database.PSQL.
	Select("id",
		"type",
		"title",
		"description",
		"attachments",
		"notifications",
		"group_id",
		"all_day",
		"repeat_type",
		"start_date",
		"end_date",
		"duration",
		"recurrence_rule",
		"exceptions",
	).
	From(database.EventsTable)
