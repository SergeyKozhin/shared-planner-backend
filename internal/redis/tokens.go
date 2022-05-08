package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/SergeyKozhin/shared-planner-backend/internal/config"
	"github.com/SergeyKozhin/shared-planner-backend/internal/model"
	"github.com/gomodule/redigo/redis"
	"github.com/xlab/closer"
	"go.uber.org/zap"
)

type RefreshTokenRepository struct {
	Pool                *redis.Pool
	SessionTTL          time.Duration
	SessionWindowPeriod time.Duration
}

func NewRefreshTokenRepository(pool *redis.Pool, logger *zap.SugaredLogger) *RefreshTokenRepository {
	sr := &RefreshTokenRepository{Pool: pool, SessionTTL: config.SessionTTl(), SessionWindowPeriod: config.SessionWindowPeriod()}

	done := make(chan bool)
	ticker := time.NewTicker(config.SessionCleanupPeriod())

	go func() {
		for {
			select {
			case <-done:
				ticker.Stop()
				return
			case <-ticker.C:
				logger.Debug("Cleaning sessions")
				err := sr.DeleteExpired(context.Background())
				if err != nil {
					logger.Errorw("error cleaning up sessions", "err", err)
				}
			}
		}
	}()

	closer.Bind(func() {
		done <- true
	})

	return sr
}

func (sr *RefreshTokenRepository) Add(ctx context.Context, session string, id int64) error {
	conn, err := sr.Pool.GetContext(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	exists, err := redis.Bool(conn.Do("EXISTS", "session:"+session))
	if err != nil {
		return err
	}
	if exists {
		return model.ErrAlreadyExists
	}

	err = conn.Send("MULTI")
	if err != nil {
		return err
	}
	err = conn.Send("SET", "session:"+session, id)
	if err != nil {
		return err
	}
	err = conn.Send("SADD", fmt.Sprintf("user:%d", id), session)
	if err != nil {
		return err
	}
	err = conn.Send("ZADD", "to_expire", time.Now().Add(sr.SessionTTL).Unix(), session)
	if err != nil {
		return err
	}
	_, err = conn.Do("EXEC")

	return err
}

func (sr *RefreshTokenRepository) Get(ctx context.Context, session string) (int64, error) {
	conn, err := sr.Pool.GetContext(ctx)
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	id, err := redis.Int64(conn.Do("GET", "session:"+session))
	if err != nil {
		if errors.Is(err, redis.ErrNil) {
			return 0, model.ErrNoRecord
		}
		return 0, err
	}

	return id, nil
}

func (sr *RefreshTokenRepository) Refresh(ctx context.Context, old, new string) error {
	conn, err := sr.Pool.GetContext(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	id, err := redis.Int64(conn.Do("GET", "session:"+old))
	if err != nil {
		if errors.Is(err, redis.ErrNil) {
			return model.ErrNoRecord
		}
		return err
	}

	exists, err := redis.Bool(conn.Do("EXISTS", "session:"+new))
	if err != nil {
		return err
	}
	if exists {
		return model.ErrAlreadyExists
	}

	err = conn.Send("MULTI")
	if err != nil {
		return err
	}

	err = conn.Send("SET", "session:"+new, id)
	if err != nil {
		return err
	}
	err = conn.Send("SADD", fmt.Sprintf("user:%d", id), new)
	if err != nil {
		return err
	}
	err = conn.Send("ZADD", "to_expire", time.Now().Add(sr.SessionTTL).Unix(), new)
	if err != nil {
		return err
	}

	err = conn.Send("ZREM", "to_expire", old)
	if err != nil {
		return err
	}
	err = conn.Send("ZADD", "to_expire", time.Now().Add(sr.SessionWindowPeriod).Unix(), old)
	if err != nil {
		return err
	}

	_, err = conn.Do("EXEC")
	return err
}

func (sr *RefreshTokenRepository) Delete(ctx context.Context, session string) error {
	conn, err := sr.Pool.GetContext(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	id, err := redis.Int64(conn.Do("GET", "session:"+session))
	if err != nil {
		if errors.Is(err, redis.ErrNil) {
			return model.ErrNoRecord
		}
		return err
	}

	err = conn.Send("MULTI")
	if err != nil {
		return err
	}
	err = conn.Send("DEL", "session:"+session)
	if err != nil {
		return err
	}
	err = conn.Send("SREM", fmt.Sprintf("user:%d", id), session)
	if err != nil {
		return err
	}
	err = conn.Send("ZREM", "to_expire", session)
	if err != nil {
		return err
	}
	_, err = conn.Do("EXEC")
	return err
}

func (sr *RefreshTokenRepository) DeleteExpired(ctx context.Context) error {
	conn, err := sr.Pool.GetContext(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	sessions, err := redis.Strings(conn.Do("ZRANGEBYSCORE", "to_expire", 0, time.Now().Unix()))
	if err != nil {
		return err
	}

	var ids []int64
	for _, session := range sessions {
		id, err := redis.Int64(conn.Do("GET", "session:"+session))
		if err != nil {
			if errors.Is(err, redis.ErrNil) {
				continue
			}
			return err
		}

		ids = append(ids, id)
	}

	err = conn.Send("MULTI")
	if err != nil {
		return err
	}

	err = conn.Send("ZREMRANGEBYSCORE", "to_expire", 0, time.Now().Unix())
	if err != nil {
		return err
	}

	for i, session := range sessions {
		err = conn.Send("DEL", "session:"+session)
		if err != nil {
			return err
		}
		err = conn.Send("SREM", fmt.Sprintf("user:%d", ids[i]), session)
		if err != nil {
			return err
		}
	}

	_, err = conn.Do("EXEC")
	if err != nil {
		return err
	}

	return nil
}

func (sr *RefreshTokenRepository) DeleteByUserID(ctx context.Context, id int64) error {
	conn, err := sr.Pool.GetContext(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	sessions, err := redis.Strings(conn.Do("SMEMBERS", fmt.Sprintf("user:%d", id)))
	if err != nil {
		return err
	}

	err = conn.Send("MULTI")
	if err != nil {
		return err
	}

	for _, session := range sessions {
		err = conn.Send("DEL", "session:"+session)
		if err != nil {
			return err
		}
		err = conn.Send("ZREM", "to_expire", session)
		if err != nil {
			return err
		}
	}

	err = conn.Send("DEL", fmt.Sprintf("user:%d", id))
	if err != nil {
		return err
	}

	_, err = conn.Do("EXEC")
	if err != nil {
		return err
	}

	return nil
}
