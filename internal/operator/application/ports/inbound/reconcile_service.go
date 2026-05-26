package inbound

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"github.com/morphy76/tacito-square/pkg/kubernetes/apis/tacito/v1alpha1"
)

// ReconcileAgentService defines the driving port interface for handling TacitoAgent reconciliation requests.
type ReconcileAgentService interface {
	Reconcile(ctx context.Context, agent *v1alpha1.TacitoAgent) error
	BuildDeployment(ctx context.Context, agent *v1alpha1.TacitoAgent) (*appsv1.Deployment, error)
	BuildHeadlessService(ctx context.Context, agent *v1alpha1.TacitoAgent) (*corev1.Service, error)
}
