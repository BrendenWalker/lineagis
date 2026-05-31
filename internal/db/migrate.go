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

// migrateAdvisoryLockKey serializes migration across concurrent integration test packages.
const migrateAdvisoryLockKey int64 = 0x766572697479 // "lineagis"

func withMigrateLock(ctx context.Context, pool *pgxpool.Pool, fn func() error) error {
	if _, err := pool.Exec(ctx, `SELECT pg_advisory_lock($1)`, migrateAdvisoryLockKey); err != nil {
		return fmt.Errorf("acquire migrate lock: %w", err)
	}
	defer func() { _, _ = pool.Exec(ctx, `SELECT pg_advisory_unlock($1)`, migrateAdvisoryLockKey) }()
	return fn()
}

func configureGoose() error {
	goose.SetBaseFS(migrations)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}
	return nil
}

// MigrateUp applies pending database migrations.
func MigrateUp(ctx context.Context, pool *pgxpool.Pool) error {
	return withMigrateLock(ctx, pool, func() error {
		db := stdlib.OpenDBFromPool(pool)
		defer func() { _ = db.Close() }()

		if err := configureGoose(); err != nil {
			return err
		}

		if err := goose.UpContext(ctx, db, "migrations"); err != nil {
			return fmt.Errorf("run migrations: %w", err)
		}

		return nil
	})
}

// MigrateDown rolls back all applied migrations.
func MigrateDown(ctx context.Context, pool *pgxpool.Pool) error {
	return withMigrateLock(ctx, pool, func() error {
		sqlDB := stdlib.OpenDBFromPool(pool)
		defer func() { _ = sqlDB.Close() }()

		if err := configureGoose(); err != nil {
			return err
		}

		if err := goose.DownToContext(ctx, sqlDB, "migrations", 0); err != nil {
			return fmt.Errorf("run migrations down: %w", err)
		}

		return nil
	})
}

// MigrateReset drops the public schema and reapplies all migrations.
// Used by integration tests; serializes with other migration callers via advisory lock.
func MigrateReset(ctx context.Context, pool *pgxpool.Pool) error {
	return withMigrateLock(ctx, pool, func() error {
		if _, err := pool.Exec(ctx, `
			DROP SCHEMA public CASCADE;
			CREATE SCHEMA public;
			GRANT ALL ON SCHEMA public TO public;
		`); err != nil {
			return fmt.Errorf("reset schema: %w", err)
		}

		db := stdlib.OpenDBFromPool(pool)
		defer func() { _ = db.Close() }()

		if err := configureGoose(); err != nil {
			return err
		}

		if err := goose.UpContext(ctx, db, "migrations"); err != nil {
			return fmt.Errorf("run migrations: %w", err)
		}

		return nil
	})
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
