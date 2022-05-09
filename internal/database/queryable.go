package database

import (
	"context"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

// PGX содержит основные операции для работы с базой данных.
type PGX interface {
	Queryable
	GetPool(ctx context.Context) *pgxpool.Pool
	BeginTx(ctx context.Context, txOptions *pgx.TxOptions) (Tx, error)
}

// Tx - транзакция
type Tx interface {
	Queryable
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

// Queryable содержит основные операции для query-инга db.
type Queryable interface {
	Exec(ctx context.Context, sqlizer sqlizer) (pgconn.CommandTag, error)
	Get(ctx context.Context, dst interface{}, sqlizer sqlizer) error
	Select(ctx context.Context, dst interface{}, sqlizer sqlizer) error
	ExecRaw(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
}

type execer interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
}

type sqlizer interface {
	ToSql() (sql string, args []interface{}, err error)
}
