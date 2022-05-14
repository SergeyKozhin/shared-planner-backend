package events

import (
	"context"

	"github.com/SergeyKozhin/shared-planner-backend/internal/database"
	"github.com/SergeyKozhin/shared-planner-backend/internal/model"
)

type Service struct {
	db               database.PGX
	eventsRepository eventsRepository
}

type eventsRepository interface {
	CreateEvent(ctx context.Context, q database.Queryable, event *model.Event) (int64, error)
	GetEventByID(ctx context.Context, q database.Queryable, id int64) (*model.Event, error)
	GetEvents(ctx context.Context, q database.Queryable, filter model.EventsFilter) ([]*model.Event, error)
	UpdateEvent(ctx context.Context, q database.Queryable, event *model.Event) error
	DeleteEvent(ctx context.Context, q database.Queryable, id int64) error
}

func NewService(db database.PGX, repo eventsRepository) *Service {
	return &Service{
		db:               db,
		eventsRepository: repo,
	}
}
