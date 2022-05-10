package api

import (
	"context"
	"io"
	"net/http"

	"github.com/SergeyKozhin/shared-planner-backend/internal/database"
	"github.com/SergeyKozhin/shared-planner-backend/internal/model"
	"github.com/SergeyKozhin/shared-planner-backend/internal/pkg/token_parser"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

type Api struct {
	handler    http.Handler
	logger     *zap.SugaredLogger
	randSource io.Reader

	jwts          jwtManager
	tokenParser   tokenParser
	refreshTokens refreshTokenRepository

	db     database.PGX
	users  userRepository
	groups groupsRepository
}

type jwtManager interface {
	CreateToken(id int64) (string, error)
	GetIdFromToken(token string) (int64, error)
}

type tokenParser interface {
	GetInfoGoogle(ctx context.Context, authCode string) (*token_parser.GoogleInfo, error)
}

type refreshTokenRepository interface {
	Add(ctx context.Context, session string, id int64) error
	Get(ctx context.Context, session string) (int64, error)
	Refresh(ctx context.Context, old, new string) error
	Delete(ctx context.Context, session string) error
	DeleteExpired(ctx context.Context) error
	DeleteByUserID(ctx context.Context, id int64) error
}

type userRepository interface {
	CreateUser(ctx context.Context, q database.Queryable, user *model.UserCreate) (int64, error)
	GetUserByEmail(ctx context.Context, q database.Queryable, email string) (*model.User, error)
	GetUserByID(ctx context.Context, q database.Queryable, id int64) (*model.User, error)
	GetUsersByIDs(ctx context.Context, q database.Queryable, ids []int64) ([]*model.User, error)
	SearchUsers(ctx context.Context, q database.Queryable, filter model.UserSearchFilter) ([]*model.User, error)
}

type groupsRepository interface {
	GetGroup(ctx context.Context, q database.Queryable, id int64) (*model.Group, error)
	GetUserGroups(ctx context.Context, q database.Queryable, userID int64) ([]*model.Group, error)
	GetUserGroupSettings(ctx context.Context, q database.Queryable, filter model.UserGroupSettingsFilter) ([]*model.GroupSettings, error)
	CreateGroup(ctx context.Context, q database.Queryable, group *model.GroupCreate) (int64, error)
	AddUserToGroup(ctx context.Context, q database.Queryable, settings *model.GroupSettings) error
	RemoveUserFromGroup(ctx context.Context, q database.Queryable, groupID int64, userID int64) error
	UpdateGroupName(ctx context.Context, q database.Queryable, groupID int64, name string) error
	UpdateGroupSettings(ctx context.Context, q database.Queryable, settings *model.GroupSettings) error
}

func NewApi(
	logger *zap.SugaredLogger,
	randSource io.Reader,
	jwts jwtManager,
	tokenParser tokenParser,
	refreshTokens refreshTokenRepository,
	db database.PGX,
	users userRepository,
	groups groupsRepository,
) (*Api, error) {
	a := &Api{
		logger:        logger,
		randSource:    randSource,
		jwts:          jwts,
		tokenParser:   tokenParser,
		refreshTokens: refreshTokens,
		db:            db,
		users:         users,
		groups:        groups,
	}
	a.setupHandler()

	return a, nil
}

func (a *Api) setupHandler() {
	middleware.DefaultLogger = func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			a.logger.Debugw(r.URL.RequestURI(),
				"addr", r.RemoteAddr,
				"protocol", r.Proto,
				"method", r.Method,
			)
			next.ServeHTTP(w, r)
		})
	}

	r := chi.NewMux()

	r.Use(middleware.Logger, middleware.Recoverer, middleware.StripSlashes)
	r.NotFound(a.notFoundResponse)
	r.MethodNotAllowed(a.methodNotAllowedResponse)

	r.Get("/healthcheck", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	r.Route("/auth", func(r chi.Router) {
		r.Post("/signin/google", a.signInGoogleHandler)
		r.Post("/refresh", a.refreshTokenHandler)
		r.Post("/logout", a.logoutUserHandler)
	})

	r.With(a.auth).Route("/", func(r chi.Router) {
		r.With(a.userCtx).Route("/user", func(r chi.Router) {
			r.Get("/", a.getUserHandler)
		})

		r.Get("/users", a.searchUsersHandler)

		r.Route("/groups", func(r chi.Router) {
			r.Get("/", a.getUserGroupsHandler)
			r.Post("/", a.createGroupHandler)
			r.With(a.groupCtx).Route("/{groupID}", func(r chi.Router) {
				r.Get("/", a.getGroupHandler)
				r.Put("/", a.updateGroupHandler)
				r.Put("/settings", a.updateGroupSettingsHandler)
			})
		})
	})

	fileServer := http.FileServer(http.Dir("./files"))
	r.Get("/files/*", http.StripPrefix("/files", fileServer).ServeHTTP)

	a.handler = r
}

func (a *Api) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.handler.ServeHTTP(w, r)
}
