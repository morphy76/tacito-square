package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/morphy76/tacito-square/internal/keeper/application/ports/outbound"
)

type txKeyType struct{}

var txKey = txKeyType{}

// TransactionRunner implements the outbound.TransactionRunner interface using PostgreSQL.
type TransactionRunner struct {
	pool *pgxpool.Pool
}

// NewTransactionRunner creates a new instance of TransactionRunner.
func NewTransactionRunner(pool *pgxpool.Pool) outbound.TransactionRunner {
	return &TransactionRunner{pool: pool}
}

// ContextWithTx injects an active pgx.Tx transaction into the context.
func ContextWithTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txKey, tx)
}

// TxFromContext retrieves an active pgx.Tx transaction from the context if it exists.
func TxFromContext(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(txKey).(pgx.Tx)
	return tx, ok
}

// PgxExecutor defines a common interface for executing SQL statements on either a pool or a transaction.
type PgxExecutor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// GetExecutor returns the active transaction from context if it exists, or falls back to the database connection pool.
func GetExecutor(ctx context.Context, pool *pgxpool.Pool) PgxExecutor {
	if tx, ok := TxFromContext(ctx); ok {
		return tx
	}
	return pool
}

// RunInTransaction executes the functional callback block within a single database transaction.
// If the callback returns an error or panics, the transaction is cleanly rolled back.
func (r *TransactionRunner) RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	// Ensure transactional cleanup in case of execution panics
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p) // Re-throw the panic after rolling back the transaction safely
		}
	}()

	txCtx := ContextWithTx(ctx, tx)
	if err := fn(txCtx); err != nil {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
			return fmt.Errorf("transaction execution error: %v, rollback failed: %w", err, rollbackErr)
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// ExecuteInTxOrPool executes the given repository logic inside the context's active transaction if present,
// or automatically manages a new local transaction if no transaction context is active.
func ExecuteInTxOrPool(ctx context.Context, pool *pgxpool.Pool, fn func(tx pgx.Tx) error) error {
	if tx, ok := TxFromContext(ctx); ok {
		return fn(tx)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin local transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := fn(tx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit local transaction: %w", err)
	}

	return nil
}
