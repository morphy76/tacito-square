//go:build integration

package postgres

import (
	"context"
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestMain(m *testing.M) {
	code := runTests(m)
	os.Exit(code)
}

func runTests(m *testing.M) int {
	ctx := context.Background()

	log.Println("Starting PostgreSQL test container via Testcontainers...")

	dbName := "tacito"
	dbUser := "tacito"
	dbPass := "tacito-dev"

	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase(dbName),
		postgres.WithUsername(dbUser),
		postgres.WithPassword(dbPass),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		log.Fatalf("Failed to start PostgreSQL container: %v", err)
	}

	defer func() {
		log.Println("Terminating PostgreSQL test container...")
		if err := testcontainers.TerminateContainer(pgContainer); err != nil {
			log.Printf("Failed to terminate container: %v", err)
		}
	}()

	// Get connection string
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		log.Fatalf("Failed to get connection string: %v", err)
	}

	log.Printf("PostgreSQL container started successfully. Connection string: %s", connStr)

	// Apply database migrations
	applyMigrations(ctx, connStr)

	// Inject the database connection URL into environment variable so tests pick it up
	os.Setenv("TS_DATABASE_URL", connStr)
	os.Setenv("TS_KEEPER_DATABASE_URL", connStr)
	os.Setenv("TS_KEEPER_MIGRATIONS_DIR", filepath.Join("..", "..", "..", "..", "..", "deploy", "postgres", "migrations"))

	// Run all package tests
	return m.Run()
}

func applyMigrations(ctx context.Context, connStr string) {
	log.Println("Applying database migrations via goose...")

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		log.Fatalf("Unable to connect to database for migrations: %v", err)
	}
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("Failed to set goose dialect: %v", err)
	}

	// Migrations are defined in deploy/postgres/migrations relative to the repository root.
	migrationsDir := filepath.Join("..", "..", "..", "..", "..", "deploy", "postgres", "migrations")
	if err := goose.Up(db, migrationsDir); err != nil {
		log.Fatalf("Goose migrations failed: %v", err)
	}

	log.Println("All database migrations applied successfully.")
}
