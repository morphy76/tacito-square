package inbound

import (
	"context"

	"github.com/morphy76/tacito-square/pkg/kubernetes/apis/tacito/v1alpha1"
)

// ReconcileAgentService defines the driving port interface for handling TacitoAgent reconciliation requests.
type ReconcileAgentService interface {
	Reconcile(ctx context.Context, agent *v1alpha1.TacitoAgent) error
}
