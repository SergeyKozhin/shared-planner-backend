package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env"
)

type config struct {
	Production           bool          `env:"PRODUCTION" envDefault:"false"`
	Port                 string        `env:"PORT" envDefault:"80"`
	PostgresUrl          string        `env:"POSTGRES_URL,required"`
	RedisUrl             string        `env:"REDIS_URL" envDefault:"redis:6379"`
	JwtTTL               time.Duration `env:"TOKEN_TTL" envDefault:"20m"`
	Secret               string        `env:"SECRET,required"`
	SessionTTl           time.Duration `env:"SESSION_TTL" envDefault:"168h"`
	SessionCleanupPeriod time.Duration `env:"SESSION_CLEANUP_PERIOD" envDefault:"60s"`
	SessionWindowPeriod  time.Duration `env:"SESSION_WINDOW_PERIOD" envDefault:"60s"`
	SessionTokenLength   int           `env:"SESSION_TOKEN_LENGTH" envDefault:"32"`
	ClientSecretPath     string        `env:"CLIENT_SECRET_PATH" envDefault:"secrets/client_secret.json"`
	RedirectURL          string        `env:"REDIRECT_URL" envDefault:""`
	ClientType           string        `env:"CLIENT_TYPE" envDefault:"web"`
	MaxFileSize          int64         `env:"MAX_FILE_SIZE" envDefault:"5242880"`
}

var conf config

func init() {
	if err := env.Parse(&conf); err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}
}

func Production() bool {
	return conf.Production
}

func Port() string {
	return conf.Port
}

func PostgresURL() string {
	return conf.PostgresUrl
}

func RedisURL() string {
	return conf.RedisUrl
}

func JwtTTL() time.Duration {
	return conf.JwtTTL
}

func Secret() string {
	return conf.Secret
}

func SessionTTl() time.Duration {
	return conf.SessionTTl
}

func SessionCleanupPeriod() time.Duration {
	return conf.SessionCleanupPeriod
}

func SessionWindowPeriod() time.Duration {
	return conf.SessionWindowPeriod
}

func SessionTokenLength() int {
	return conf.SessionTokenLength
}

func ClientSecretPath() string {
	return conf.ClientSecretPath
}

func RedirectURL() string {
	return conf.RedirectURL
}

func MaxFileSize() int64 {
	return conf.MaxFileSize
}

func ClientType() string {
	return conf.ClientType
}
