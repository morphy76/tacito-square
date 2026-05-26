package inbound

import (
	"context"

	"github.com/morphy76/tacito-square/internal/operator/application/ports/inbound"
	"github.com/morphy76/tacito-square/pkg/kubernetes/apis/tacito/v1alpha1"
	"github.com/rs/zerolog"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TacitoAgentReconciler reconciles a TacitoAgent object.
type TacitoAgentReconciler struct {
	client  client.Client
	scheme  *runtime.Scheme
	service inbound.ReconcileAgentService
	logger  zerolog.Logger
}

// NewTacitoAgentReconciler constructs a new TacitoAgentReconciler.
func NewTacitoAgentReconciler(
	client client.Client,
	scheme *runtime.Scheme,
	service inbound.ReconcileAgentService,
	logger zerolog.Logger,
) *TacitoAgentReconciler {
	return &TacitoAgentReconciler{
		client:  client,
		scheme:  scheme,
		service: service,
		logger:  logger,
	}
}

// Reconcile handles a TacitoAgent reconciliation request.
func (r *TacitoAgentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// First fetch the TacitoAgent resource
	agent := &v1alpha1.TacitoAgent{}
	err := r.client.Get(ctx, req.NamespacedName, agent)
	if err != nil {
		if apierrors.IsNotFound(err) {
			r.logger.Info().
				Str("namespace", req.Namespace).
				Str("name", req.Name).
				Msg("TacitoAgent resource not found, ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		r.logger.Error().Err(err).
			Str("namespace", req.Namespace).
			Str("name", req.Name).
			Msg("failed to get TacitoAgent resource")
		return ctrl.Result{}, err
	}

	// Delegate to our application service
	err = r.service.Reconcile(ctx, agent)
	if err != nil {
		r.logger.Error().Err(err).
			Str("namespace", req.Namespace).
			Str("name", req.Name).
			Msg("failed to reconcile TacitoAgent through application service")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TacitoAgentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.TacitoAgent{}).
		Complete(r)
}
