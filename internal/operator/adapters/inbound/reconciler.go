package inbound

import (
	"context"
	"time"

	"github.com/morphy76/tacito-square/internal/operator/application/ports/inbound"
	"github.com/morphy76/tacito-square/pkg/kubernetes/apis/tacito/v1alpha1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	reconciliationTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tacito_operator_reconciliation_total",
			Help: "Total number of TacitoAgent reconciliation executions",
		},
		[]string{"status"},
	)
	reconciliationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "tacito_operator_reconciliation_duration_seconds",
			Help:    "Duration of TacitoAgent reconciliation executions in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"status"},
	)
	activeAgents = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "tacito_operator_active_agents",
			Help: "Current number of active TacitoAgent resources",
		},
	)
)

func init() {
	prometheus.MustRegister(reconciliationTotal)
	prometheus.MustRegister(reconciliationDuration)
	prometheus.MustRegister(activeAgents)

	// Pre-initialize vector metrics series so they appear immediately in GET /metrics scrapes
	reconciliationTotal.WithLabelValues("success")
	reconciliationTotal.WithLabelValues("error")
	reconciliationDuration.WithLabelValues("success")
	reconciliationDuration.WithLabelValues("error")
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
		reconciliationTotal.WithLabelValues("error").Inc()
		reconciliationDuration.WithLabelValues("error").Observe(duration)
		return ctrl.Result{}, err
	}

	reconciliationTotal.WithLabelValues("success").Inc()
	reconciliationDuration.WithLabelValues("success").Observe(duration)
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
		activeAgents.Set(float64(len(list.Items)))
	}
}
