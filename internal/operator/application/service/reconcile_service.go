package service

import (
	"context"
	"fmt"
	"os"

	"github.com/morphy76/tacito-square/internal/operator/application/ports/inbound"
	"github.com/morphy76/tacito-square/pkg/kubernetes/apis/tacito/v1alpha1"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// LoadTierMap loads and parses agent tier profiles from a YAML file.
func LoadTierMap(path string, logger zerolog.Logger) (map[string]TierProfile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Warn().Str("path", path).Msg("tier configuration file does not exist, using empty tier map")
			return make(map[string]TierProfile), nil
		}
		return nil, fmt.Errorf("failed to read tier configuration file: %w", err)
	}

	var tierMap map[string]TierProfile
	if err := yaml.Unmarshal(data, &tierMap); err != nil {
		return nil, fmt.Errorf("failed to parse tier configuration YAML: %w", err)
	}

	logger.Info().Int("count", len(tierMap)).Str("path", path).Msg("successfully loaded agent tier profiles")
	return tierMap, nil
}

// ReconcileAgentServiceImpl implements the ReconcileAgentService use case.
type ReconcileAgentServiceImpl struct {
	client  client.Client
	logger  zerolog.Logger
	cfg     *viper.Viper
	tierMap map[string]TierProfile
}

var _ inbound.ReconcileAgentService = (*ReconcileAgentServiceImpl)(nil)

// NewReconcileAgentService constructs a new ReconcileAgentServiceImpl.
func NewReconcileAgentService(
	c client.Client,
	logger zerolog.Logger,
	cfg *viper.Viper,
	tierMap map[string]TierProfile,
) *ReconcileAgentServiceImpl {
	return &ReconcileAgentServiceImpl{
		client:  c,
		logger:  logger,
		cfg:     cfg,
		tierMap: tierMap,
	}
}

// Reconcile coordinates the reconciliation of a TacitoAgent.
func (s *ReconcileAgentServiceImpl) Reconcile(ctx context.Context, agent *v1alpha1.TacitoAgent) error {
	s.logger.Debug().
		Str("namespace", agent.Namespace).
		Str("name", agent.Name).
		Str("tenant_id", agent.Spec.TenantID).
		Str("agent_name", agent.Spec.AgentName).
		Msg("entering Reconcile: reconciling tacito agent resource")

	// 1. Reconcile the Deployment
	existingDep := &appsv1.Deployment{}
	err := s.client.Get(ctx, client.ObjectKey{Namespace: agent.Namespace, Name: agent.Name}, existingDep)
	if err != nil {
		if apierrors.IsNotFound(err) {
			dep, buildErr := s.BuildDeployment(ctx, agent)
			if buildErr != nil {
				s.logger.Error().Err(buildErr).Msg("failed to build Deployment")
				return buildErr
			}
			if createErr := s.client.Create(ctx, dep); createErr != nil {
				s.logger.Error().Err(createErr).Msg("failed to create Deployment")
				return createErr
			}
			s.logger.Info().Str("namespace", agent.Namespace).Str("name", agent.Name).Msg("Deployment created successfully")
			existingDep = dep
		} else {
			s.logger.Error().Err(err).Msg("failed to get Deployment")
			return err
		}
	} else {
		dep, buildErr := s.BuildDeployment(ctx, agent)
		if buildErr != nil {
			s.logger.Error().Err(buildErr).Msg("failed to build Deployment for update")
			return buildErr
		}
		existingDep.Spec = dep.Spec
		existingDep.Labels = dep.Labels
		if updateErr := s.client.Update(ctx, existingDep); updateErr != nil {
			s.logger.Error().Err(updateErr).Msg("failed to update Deployment")
			return updateErr
		}
		s.logger.Info().Str("namespace", agent.Namespace).Str("name", agent.Name).Msg("Deployment updated successfully")
	}

	// 2. Reconcile the headless Service
	existingSvc := &corev1.Service{}
	err = s.client.Get(ctx, client.ObjectKey{Namespace: agent.Namespace, Name: agent.Name}, existingSvc)
	if err != nil {
		if apierrors.IsNotFound(err) {
			svc, buildErr := s.BuildHeadlessService(ctx, agent)
			if buildErr != nil {
				s.logger.Error().Err(buildErr).Msg("failed to build headless Service")
				return buildErr
			}
			if createErr := s.client.Create(ctx, svc); createErr != nil {
				s.logger.Error().Err(createErr).Msg("failed to create headless Service")
				return createErr
			}
			s.logger.Info().Str("namespace", agent.Namespace).Str("name", agent.Name).Msg("headless Service created successfully")
		} else {
			s.logger.Error().Err(err).Msg("failed to get headless Service")
			return err
		}
	} else {
		svc, buildErr := s.BuildHeadlessService(ctx, agent)
		if buildErr != nil {
			s.logger.Error().Err(buildErr).Msg("failed to build headless Service for update")
			return buildErr
		}
		clusterIP := existingSvc.Spec.ClusterIP
		existingSvc.Spec = svc.Spec
		existingSvc.Spec.ClusterIP = clusterIP
		existingSvc.Labels = svc.Labels
		if updateErr := s.client.Update(ctx, existingSvc); updateErr != nil {
			s.logger.Error().Err(updateErr).Msg("failed to update headless Service")
			return updateErr
		}
		s.logger.Info().Str("namespace", agent.Namespace).Str("name", agent.Name).Msg("headless Service updated successfully")
	}

	// 3. Check Deployment ready replicas and update TacitoAgent status
	readyReplicas := existingDep.Status.ReadyReplicas
	agent.Status.Replicas = readyReplicas

	var phase v1alpha1.TacitoAgentPhase
	var condStatus metav1.ConditionStatus
	var reason, message string

	if readyReplicas > 0 {
		phase = v1alpha1.PhaseRunning
		condStatus = metav1.ConditionTrue
		reason = "MinimumReplicasAvailable"
		message = fmt.Sprintf("Agent deployment has %d ready replica(s).", readyReplicas)
	} else {
		if agent.Spec.Replicas != nil && *agent.Spec.Replicas == 0 {
			phase = v1alpha1.PhaseIdle
			condStatus = metav1.ConditionFalse
			reason = "ScaleToZero"
			message = "Agent has been scaled to zero replicas."
		} else {
			phase = v1alpha1.PhasePending
			condStatus = metav1.ConditionFalse
			reason = "NoReplicasAvailable"
			message = "Agent is waiting for ready replicas."
		}
	}

	agent.Status.Phase = phase

	// Update "Available" condition
	var found bool
	for i, cond := range agent.Status.Conditions {
		if cond.Type == "Available" {
			agent.Status.Conditions[i].Status = condStatus
			agent.Status.Conditions[i].Reason = reason
			agent.Status.Conditions[i].Message = message
			agent.Status.Conditions[i].LastTransitionTime = metav1.Now()
			found = true
			break
		}
	}
	if !found {
		agent.Status.Conditions = append(agent.Status.Conditions, metav1.Condition{
			Type:               "Available",
			Status:             condStatus,
			Reason:             reason,
			Message:            message,
			LastTransitionTime: metav1.Now(),
		})
	}

	// Update agent status subresource
	if statusErr := s.client.Status().Update(ctx, agent); statusErr != nil {
		s.logger.Error().Err(statusErr).Msg("failed to update agent status")
		return statusErr
	}
	s.logger.Info().Str("namespace", agent.Namespace).Str("name", agent.Name).Msg("agent status updated successfully")

	s.logger.Info().
		Str("namespace", agent.Namespace).
		Str("name", agent.Name).
		Str("tenant_id", agent.Spec.TenantID).
		Str("agent_name", agent.Spec.AgentName).
		Msg("reconciled tacito agent resource successfully")

	return nil
}



