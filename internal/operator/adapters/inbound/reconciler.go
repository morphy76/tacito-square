package inbound

import (
	"context"
	"time"

	"github.com/morphy76/tacito-square/internal/operator/application/ports/inbound"
	"github.com/morphy76/tacito-square/pkg/kubernetes/apis/tacito/v1alpha1"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
	"github.com/rs/zerolog"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	meter = otel.Meter("operator-reconciler")

	reconciliationTotal, _ = meter.Int64Counter(
		"tacito_operator_reconciliation_total",
		otelmetric.WithDescription("Total number of TacitoAgent reconciliation executions"),
	)
	reconciliationDuration, _ = meter.Float64Histogram(
		"tacito_operator_reconciliation_duration_seconds",
		otelmetric.WithDescription("Duration of TacitoAgent reconciliation executions in seconds"),
	)
	activeAgents, _ = meter.Int64Gauge(
		"tacito_operator_active_agents",
		otelmetric.WithDescription("Current number of active TacitoAgent resources"),
	)
)

// InitReconcilerMetrics re-initializes and registers reconciler metrics with the concrete MeterProvider.
func InitReconcilerMetrics() {
	meter = otel.Meter("operator-reconciler")

	reconciliationTotal, _ = meter.Int64Counter(
		"tacito_operator_reconciliation_total",
		otelmetric.WithDescription("Total number of TacitoAgent reconciliation executions"),
	)
	reconciliationDuration, _ = meter.Float64Histogram(
		"tacito_operator_reconciliation_duration_seconds",
		otelmetric.WithDescription("Duration of TacitoAgent reconciliation executions in seconds"),
	)
	activeAgents, _ = meter.Int64Gauge(
		"tacito_operator_active_agents",
		otelmetric.WithDescription("Current number of active TacitoAgent resources"),
	)

	// Pre-initialize vector metrics series so they appear immediately in GET /metrics scrapes
	reconciliationTotal.Add(context.Background(), 0, otelmetric.WithAttributes(attribute.String("status", "success")))
	reconciliationTotal.Add(context.Background(), 0, otelmetric.WithAttributes(attribute.String("status", "error")))
	reconciliationDuration.Record(context.Background(), 0, otelmetric.WithAttributes(attribute.String("status", "success")))
	reconciliationDuration.Record(context.Background(), 0, otelmetric.WithAttributes(attribute.String("status", "error")))
	activeAgents.Record(context.Background(), 0)
}

func init() {
	// Pre-initialize vector metrics series so they appear immediately in GET /metrics scrapes
	reconciliationTotal.Add(context.Background(), 0, otelmetric.WithAttributes(attribute.String("status", "success")))
	reconciliationTotal.Add(context.Background(), 0, otelmetric.WithAttributes(attribute.String("status", "error")))
	reconciliationDuration.Record(context.Background(), 0, otelmetric.WithAttributes(attribute.String("status", "success")))
	reconciliationDuration.Record(context.Background(), 0, otelmetric.WithAttributes(attribute.String("status", "error")))
	activeAgents.Record(context.Background(), 0)
}

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
	start := time.Now()

	// First fetch the TacitoAgent resource
	agent := &v1alpha1.TacitoAgent{}
	err := r.client.Get(ctx, req.NamespacedName, agent)
	if err != nil {
		if apierrors.IsNotFound(err) {
			r.logger.Info().
				Str("namespace", req.Namespace).
				Str("name", req.Name).
				Msg("TacitoAgent resource not found, ignoring since object must be deleted")
			r.updateActiveAgentsMetric(ctx)
			return ctrl.Result{}, nil
		}
		r.logger.Error().Err(err).
			Str("namespace", req.Namespace).
			Str("name", req.Name).
			Msg("failed to get TacitoAgent resource")
		return ctrl.Result{}, err
	}

	r.updateActiveAgentsMetric(ctx)

	// Delegate to our application service
	err = r.service.Reconcile(ctx, agent)
	duration := time.Since(start).Seconds()
	if err != nil {
		r.logger.Error().Err(err).
			Str("namespace", req.Namespace).
			Str("name", req.Name).
			Msg("failed to reconcile TacitoAgent through application service")
		reconciliationTotal.Add(ctx, 1, otelmetric.WithAttributes(attribute.String("status", "error")))
		reconciliationDuration.Record(ctx, duration, otelmetric.WithAttributes(attribute.String("status", "error")))
		return ctrl.Result{}, err
	}

	reconciliationTotal.Add(ctx, 1, otelmetric.WithAttributes(attribute.String("status", "success")))
	reconciliationDuration.Record(ctx, duration, otelmetric.WithAttributes(attribute.String("status", "success")))
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TacitoAgentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.TacitoAgent{}).
		Complete(r)
}

func (r *TacitoAgentReconciler) updateActiveAgentsMetric(ctx context.Context) {
	var list v1alpha1.TacitoAgentList
	if err := r.client.List(ctx, &list); err == nil {
		activeAgents.Record(ctx, int64(len(list.Items)))
	}
}
