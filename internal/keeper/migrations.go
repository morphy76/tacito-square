package keeper

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/rs/zerolog"
)

// RunMigrations runs database migrations using the approved goose framework.
func RunMigrations(ctx context.Context, cfg *pgxpool.Config, migrationsDir string, logger zerolog.Logger) error {
	if migrationsDir == "" {
		migrationsDir = os.Getenv("TS_KEEPER_MIGRATIONS_DIR")
		if migrationsDir == "" {
			migrationsDir = "deploy/postgres/migrations"
		}
	}

	logger.Debug().Str("dir", migrationsDir).Msg("starting database migrations via goose")

	// Standard sql.DB opened using the exact pgx.ConnConfig parsed at boot time
	var db *sql.DB = stdlib.OpenDB(*cfg.ConnConfig)
	defer db.Close()

	// Configure goose to use postgres dialect
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	// Apply database migrations Up to the latest version
	if err := goose.Up(db, migrationsDir); err != nil {
		return fmt.Errorf("goose database migrations failed: %w", err)
	}

	logger.Debug().Msg("all database migrations applied successfully via goose")
	return nil
}
