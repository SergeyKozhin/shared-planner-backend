package main

import (
	"context"
	"crypto/rand"
	"log"
	"net/http"

	"github.com/SergeyKozhin/shared-planner-backend/internal/api"
	events_service "github.com/SergeyKozhin/shared-planner-backend/internal/business/events"
	"github.com/SergeyKozhin/shared-planner-backend/internal/config"
	_ "github.com/SergeyKozhin/shared-planner-backend/internal/config"
	"github.com/SergeyKozhin/shared-planner-backend/internal/database"
	"github.com/SergeyKozhin/shared-planner-backend/internal/database/events"
	"github.com/SergeyKozhin/shared-planner-backend/internal/database/group"
	"github.com/SergeyKozhin/shared-planner-backend/internal/database/user"
	"github.com/SergeyKozhin/shared-planner-backend/internal/notifications"
	"github.com/SergeyKozhin/shared-planner-backend/internal/pkg/fcm"
	"github.com/SergeyKozhin/shared-planner-backend/internal/pkg/jwt"
	"github.com/SergeyKozhin/shared-planner-backend/internal/pkg/oauth"
	"github.com/SergeyKozhin/shared-planner-backend/internal/redis"
	"github.com/xlab/closer"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	ctx := context.Background()

	logger, err := initLogger()
	if err != nil {
		log.Fatalf("unable to initializae logger: %v", err)
	}

	jwts := jwt.NewManger()
	tokenParser := oauth.NewParser()

	redisPool := redis.NewRedisPool(logger)
	refreshTokens := redis.NewRefreshTokenRepository(redisPool, logger)

	db, err := database.NewPGX(ctx)
	if err != nil {
		log.Fatalf("unable to initializae db: %v", err)
	}
	usersRepository := user.NewRepository()
	groupsRepository := group.NewRepository()
	eventsRepository := events.NewRepository()

	eventsService := events_service.NewService(db, eventsRepository)

	fcmService, err := fcm.NewService(ctx)
	if err != nil {
		log.Fatalf("unable to initializae fcm service: %v", err)
	}

	sender := notifications.NewSender(db, logger, groupsRepository, usersRepository, eventsService, fcmService)
	go sender.Start(ctx)

	api, err := api.NewApi(
		logger,
		rand.Reader,
		jwts,
		tokenParser,
		refreshTokens,
		db,
		usersRepository,
		groupsRepository,
		eventsService,
		fcmService,
	)

	errLogger, err := zap.NewStdLogAt(logger.Desugar(), zap.ErrorLevel)
	if err != nil {
		logger.Fatalw("error initiating server logger", "err", err)
	}

	server := &http.Server{
		Addr:     ":" + config.Port(),
		Handler:  api,
		ErrorLog: errLogger,
	}

	logger.Infow("Started server", "port", config.Port())
	logger.Fatalw("server error", "err", server.ListenAndServe())
}

func initLogger() (*zap.SugaredLogger, error) {
	var logger *zap.Logger
	var err error

	if config.Production() {
		logger, err = zap.NewProduction()
	} else {
		conf := zap.NewDevelopmentConfig()
		conf.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		logger, err = conf.Build()
	}

	if err != nil {
		return nil, err
	}

	closer.Bind(func() {
		_ = logger.Sync()
	})

	return logger.Sugar(), nil
}
