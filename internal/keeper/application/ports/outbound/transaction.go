package outbound

import "context"

// TransactionRunner orchestrates transactional boundaries safely across hexagonal layers.
// This interface allows application use case orchestrators to start database transactions
// using context propagation without importing specific database client libraries.
type TransactionRunner interface {
	RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
