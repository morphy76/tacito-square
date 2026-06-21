package outbound

import "context"

// KeeperClient defines the outbound port for interacting with the Keeper core services.
type KeeperClient interface {
	// Ping checks the health/connectivity of the Keeper service.
	Ping(ctx context.Context) error
}
