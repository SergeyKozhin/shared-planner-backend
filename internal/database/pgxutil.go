package database

import (
	"context"
	"fmt"

	"github.com/SergeyKozhin/shared-planner-backend/internal/config"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/xlab/closer"
)

// pgxUtil обертка для упрощенной работы с pgx.
type pgxUtil struct {
	pool *pgxpool.Pool
}

// NewPGX создает структуру, с помощью которой получается доступ к pgx pool
func NewPGX(ctx context.Context) (PGX, error) {
	pool, err := pgxpool.Connect(ctx, config.PostgresURL())
	if err != nil {
		return nil, err
	}

	closer.Bind(pool.Close)

	return &pgxUtil{pool: pool}, nil
}

// BeginTx транзакцию.
func (p *pgxUtil) BeginTx(ctx context.Context, txOptions *pgx.TxOptions) (Tx, error) {
	var txOpts pgx.TxOptions
	if txOptions != nil {
		txOpts = *txOptions
	}

	tx, err := p.pool.BeginTx(ctx, txOpts)
	if err != nil {
		return nil, fmt.Errorf("не удалось начать транзакцию: %w", err)
	}

	return &txUtil{pgxTx: tx}, nil
}

// ExecRaw исполняет query.
func (p *pgxUtil) ExecRaw(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error) {
	return p.pool.Exec(ctx, sql, arguments...)
}

// Exec исполняет query.
func (p *pgxUtil) Exec(ctx context.Context, sqlizer sqlizer) (pgconn.CommandTag, error) {
	return execFn(ctx, p.pool, sqlizer)
}

// Select может сканировать сразу несколько рядов в slice.
// Если рядов нет, возвращает nil.
func (p *pgxUtil) Select(ctx context.Context, dst interface{}, sqlizer sqlizer) error {
	return selectFn(ctx, p.pool, dst, sqlizer)
}

// Get сканирует один ряд.
// Если рядов нет, возвращает ошибку pgx.ErrNoRows.
func (p *pgxUtil) Get(ctx context.Context, dst interface{}, sqlizer sqlizer) error {
	return getFn(ctx, p.pool, dst, sqlizer)
}

// GetPool получает и возвращает pool.
func (p *pgxUtil) GetPool(ctx context.Context) *pgxpool.Pool {
	return p.pool
}

// Tx обертка над транзакцией.
type txUtil struct {
	pgxTx pgx.Tx
}

func (t *txUtil) ExecRaw(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error) {
	return t.pgxTx.Exec(ctx, sql, arguments...)
}

// Exec исполняет query.
func (t *txUtil) Exec(ctx context.Context, sqlizer sqlizer) (pgconn.CommandTag, error) {
	return execFn(ctx, t.pgxTx, sqlizer)
}

// Select может сканировать сразу несколько рядов в slice.
// Если рядов нет, возвращает nil.
func (t *txUtil) Select(ctx context.Context, dst interface{}, sqlizer sqlizer) error {
	return selectFn(ctx, t.pgxTx, dst, sqlizer)
}

// Get сканирует один ряд.
// Если рядов нет, возвращает ошибку pgx.ErrNoRows.
func (t *txUtil) Get(ctx context.Context, dst interface{}, sqlizer sqlizer) error {
	return getFn(ctx, t.pgxTx, dst, sqlizer)
}

// Commit завершает транзакцию.
func (t *txUtil) Commit(ctx context.Context) error {
	return t.pgxTx.Commit(ctx)
}

// Rollback откатывает транзакцию.
func (t *txUtil) Rollback(ctx context.Context) error {
	return t.pgxTx.Rollback(ctx)
}

func execFn(ctx context.Context, e execer, sqlizer sqlizer) (pgconn.CommandTag, error) {
	query, args, err := sqlizer.ToSql()
	if err != nil {
		return nil, fmt.Errorf("ToSql: %w", err)
	}

	return e.Exec(ctx, query, args...)
}

func selectFn(ctx context.Context, q pgxscan.Querier, dst interface{}, sqlizer sqlizer) error {
	query, args, err := sqlizer.ToSql()
	if err != nil {
		return fmt.Errorf("ToSql: %w", err)
	}

	return pgxscan.Select(ctx, q, dst, query, args...)
}

func getFn(ctx context.Context, q pgxscan.Querier, dst interface{}, sqlizer sqlizer) error {
	query, args, err := sqlizer.ToSql()
	if err != nil {
		return fmt.Errorf("ToSql: %w", err)
	}

	return pgxscan.Get(ctx, q, dst, query, args...)
}
