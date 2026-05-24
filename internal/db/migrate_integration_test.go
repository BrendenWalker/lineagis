//go:build integration

package db_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/BrendenWalker/verity/internal/db"
)

func TestMigrateUpDown(t *testing.T) {
	url := os.Getenv("VERITY_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("VERITY_TEST_DATABASE_URL not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pool, err := db.OpenPool(ctx, url)
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}
	defer pool.Close()

	if err := db.MigrateReset(ctx, pool); err != nil {
		t.Fatalf("migrate reset: %v", err)
	}

	var tableExists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'namespaces'
		)
	`).Scan(&tableExists)
	if err != nil {
		t.Fatalf("check namespaces table: %v", err)
	}
	if !tableExists {
		t.Fatal("namespaces table should exist after migrate up")
	}

	if err := db.MigrateDown(ctx, pool); err != nil {
		t.Fatalf("migrate down: %v", err)
	}

	err = pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'namespaces'
		)
	`).Scan(&tableExists)
	if err != nil {
		t.Fatalf("check namespaces gone: %v", err)
	}
	if tableExists {
		t.Fatal("namespaces table should not exist after migrate down")
	}

	if err := db.MigrateUp(ctx, pool); err != nil {
		t.Fatalf("migrate up again: %v", err)
	}
}
