//go:build integration

package postgres

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
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

	// Run all package tests
	return m.Run()
}

func applyMigrations(ctx context.Context, connStr string) {
	log.Println("Applying database migrations...")

	// Connect using pgx directly to run migrations
	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		log.Fatalf("Unable to connect to database for migrations: %v", err)
	}
	defer conn.Close(ctx)

	// Migrations are defined in deploy/postgres/migrations relative to the repository root.
	// Since main_test.go is inside internal/keeper/adapters/outbound/postgres,
	// the path to deploy/postgres/migrations is ../../../../../deploy/postgres/migrations.
	migrationsDir := filepath.Join("..", "..", "..", "..", "..", "deploy", "postgres", "migrations")

	files, err := ioutil.ReadDir(migrationsDir)
	if err != nil {
		log.Fatalf("Failed to read migrations directory: %v", err)
	}

	var filenames []string
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".sql") {
			filenames = append(filenames, f.Name())
		}
	}
	sort.Strings(filenames)

	for _, fname := range filenames {
		path := filepath.Join(migrationsDir, fname)
		contentBytes, err := ioutil.ReadFile(path)
		if err != nil {
			log.Fatalf("Failed to read migration file %s: %v", fname, err)
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

		log.Printf("Executing migration: %s\n", fname)
		_, err = conn.Exec(ctx, upSql)
		if err != nil {
			log.Fatalf("Migration %s failed: %v\nSQL:\n%s", fname, err, upSql)
		}
	}

	log.Println("All database migrations applied successfully.")
}
