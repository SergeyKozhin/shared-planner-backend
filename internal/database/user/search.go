package user

import (
	"context"
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/SergeyKozhin/shared-planner-backend/internal/database"
	"github.com/SergeyKozhin/shared-planner-backend/internal/model"
)

func (*Repository) SearchUsers(ctx context.Context, q database.Queryable, filter model.UserSearchFilter) ([]*model.User, error) {
	query := fmt.Sprintf("%%%v%%", strings.Join(strings.Split(filter.Query, " "), "%"))

	qb := baseQuery.
		Where(sq.ILike{"full_name || ' ' || email || ' ' || phone_number": query}).
		Limit(uint64(filter.Limit)).
		Offset(uint64((filter.Page-1)*filter.Limit)).
		OrderByClause("full_name || ' ' || email || ' ' || phone_number <-> ?", query)

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
