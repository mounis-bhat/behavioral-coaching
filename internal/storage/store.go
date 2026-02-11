package storage

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mounis-bhat/starter/internal/config"
	"github.com/mounis-bhat/starter/internal/storage/db"
)

type Store struct {
	pool    *pgxpool.Pool
	Queries *db.Queries
}

func New(ctx context.Context, cfg config.DatabaseConfig) (*Store, error) {
	pool, err := pgxpool.New(ctx, cfg.ConnectionString())
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if err := ensureMigrationsApplied(ctx, pool); err != nil {
		pool.Close()
		return nil, err
	}

	return &Store{
		pool:    pool,
		Queries: db.New(pool),
	}, nil
}

func ensureMigrationsApplied(ctx context.Context, pool *pgxpool.Pool) error {
	if skipMigrationCheck() {
		return nil
	}

	const migrationsTable = "goose_db_version"

	var exists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = $1
		)
	`, migrationsTable).Scan(&exists); err != nil {
		return fmt.Errorf("failed to verify migrations: %w", err)
	}

	if !exists {
		return fmt.Errorf("database is not migrated (missing %s). Run `make migrate-up`", migrationsTable)
	}

	var count int
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM goose_db_version").Scan(&count); err != nil {
		return fmt.Errorf("failed to read %s: %w", migrationsTable, err)
	}

	if count == 0 {
		return fmt.Errorf("database has no applied migrations. Run `make migrate-up`")
	}

	return nil
}

func skipMigrationCheck() bool {
	value := os.Getenv("SKIP_MIGRATION_CHECK")
	if value == "" {
		return false
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false
	}
	return parsed
}

func (s *Store) Close() {
	if s.pool != nil {
		s.pool.Close()
	}
}

func (s *Store) Pool() *pgxpool.Pool {
	return s.pool
}
