package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrations embed.FS

// MigrateUp applies pending database migrations.
func MigrateUp(ctx context.Context, pool *pgxpool.Pool) error {
	db := stdlib.OpenDBFromPool(pool)
	defer func() { _ = db.Close() }()

	goose.SetBaseFS(migrations)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}

	if err := goose.UpContext(ctx, db, "migrations"); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}

// Ping verifies database connectivity.
func Ping(ctx context.Context, pool *pgxpool.Pool) error {
	var one int
	if err := pool.QueryRow(ctx, "SELECT 1").Scan(&one); err != nil {
		return err
	}
	if one != 1 {
		return fmt.Errorf("unexpected ping result: %d", one)
	}
	return nil
}

// OpenSQL returns a *sql.DB backed by the pool for tests.
func OpenSQL(pool *pgxpool.Pool) *sql.DB {
	return stdlib.OpenDBFromPool(pool)
}
