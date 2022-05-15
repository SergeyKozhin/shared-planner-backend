package redis

import (
	"context"

	"github.com/SergeyKozhin/shared-planner-backend/internal/config"
	"github.com/gomodule/redigo/redis"
	"github.com/xlab/closer"
	"go.uber.org/zap"
)

func NewRedisPool(logger *zap.SugaredLogger) *redis.Pool {
	pool := &redis.Pool{
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", config.RedisURL())
		},
		DialContext: func(ctx context.Context) (redis.Conn, error) {
			return redis.DialContext(ctx, "tcp", config.RedisURL())
		},
	}

	closer.Bind(func() {
		if err := pool.Close(); err != nil {
			logger.Errorw("Failed closing redis pool", "err", err)
		}
	})

	return pool
}
