package keeper

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// RunMigrations runs database migrations using raw pgx connections.
func RunMigrations(ctx context.Context, cfg *pgxpool.Config, migrationsDir string, logger zerolog.Logger) error {
	if migrationsDir == "" {
		migrationsDir = os.Getenv("TS_KEEPER_MIGRATIONS_DIR")
		if migrationsDir == "" {
			migrationsDir = "deploy/postgres/migrations"
		}
	}

	logger.Info().Str("dir", migrationsDir).Msg("starting database migrations")

	conn, err := pgx.ConnectConfig(ctx, cfg.ConnConfig)
	if err != nil {
		if strings.Contains(err.Error(), "server refused TLS connection") {
			logger.Warn().
				Msg("DATABASE CONNECTION DIAGNOSIS: The PostgreSQL server refused the TLS connection request. " +
					"This typically means that SSL/TLS is disabled on the server (e.g. 'ssl = off' in postgresql.conf). " +
					"To bypass this in development, modify your connection URL by setting 'sslmode=prefer' or 'sslmode=disable' (e.g., TS_KEEPER_DATABASE_URL=\"postgres://...sslmode=disable\"). " +
					"In production, verify that the PostgreSQL server has TLS enabled and correct certificates are loaded.")
		}
		return fmt.Errorf("unable to connect to database: %w", err)
	}
	defer conn.Close(ctx)

	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var filenames []string
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".sql") {
			filenames = append(filenames, f.Name())
		}
	}
	sort.Strings(filenames)

	for _, fname := range filenames {
		path := filepath.Join(migrationsDir, fname)
		contentBytes, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", fname, err)
		}
		content := string(contentBytes)

		// Parse the UP migration SQL block (before "-- +goose Down" comment)
		var upSql string
		downIndex := strings.Index(content, "-- +goose Down")
		if downIndex != -1 {
			upSql = content[:downIndex]
		} else {
			upSql = content
		}

		// Clean up goose directive lines
		upSql = strings.ReplaceAll(upSql, "-- +goose Up", "")
		upSql = strings.ReplaceAll(upSql, "-- +goose StatementBegin", "")
		upSql = strings.ReplaceAll(upSql, "-- +goose StatementEnd", "")

		logger.Info().Str("file", fname).Msg("executing database migration")
		_, err = conn.Exec(ctx, upSql)
		if err != nil {
			return fmt.Errorf("migration %s failed: %w\nSQL:\n%s", fname, err, upSql)
		}
	}

	logger.Info().Msg("all database migrations applied successfully")
	return nil
}
