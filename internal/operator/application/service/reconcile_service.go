package service

import (
	"context"
	"fmt"
	"strconv"

	"github.com/morphy76/tacito-square/internal/operator/application/ports/inbound"
	"github.com/morphy76/tacito-square/pkg/kubernetes/apis/tacito/v1alpha1"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultAgentImage    = "tacito-square/agent:0.1.0"
	defaultAgentLogLevel = "info"
	defaultAgentPort     = "8081"
	defaultNatsURL       = "nats://tacito-infra-nats:4222"
	defaultRedisURL      = "rediss://tacito-infra-redis:6379"
	defaultQdrantURL     = "http://tacito-infra-qdrant:6334"
	defaultOtelEndpoint  = "tacito-infra-otel-collector:4317"
	defaultS3Endpoint    = "https://tacito-infra-minio:9000"
	defaultS3Bucket      = "tacito"
	defaultS3Region      = "us-east-1"
	defaultCaCertPath    = "/etc/ssl/tacito/ca.crt"
)

// ReconcileAgentServiceImpl implements the ReconcileAgentService use case.
type ReconcileAgentServiceImpl struct {
	client client.Client
	logger zerolog.Logger
	cfg    *viper.Viper
}

var _ inbound.ReconcileAgentService = (*ReconcileAgentServiceImpl)(nil)

// NewReconcileAgentService constructs a new ReconcileAgentServiceImpl.
func NewReconcileAgentService(c client.Client, logger zerolog.Logger, cfg *viper.Viper) *ReconcileAgentServiceImpl {
	return &ReconcileAgentServiceImpl{
		client: c,
		logger: logger,
		cfg:    cfg,
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
			existingSvc = svc
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

// BuildDeployment constructs the Kubernetes Deployment specification for the agent.
func (s *ReconcileAgentServiceImpl) BuildDeployment(ctx context.Context, agent *v1alpha1.TacitoAgent) (*appsv1.Deployment, error) {
	s.logger.Debug().Str("namespace", agent.Namespace).Str("name", agent.Name).Msg("entering BuildDeployment")
	// 1. Resolve configurations from Viper with proper fallbacks
	image := s.getAgentSetting("agent.image", defaultAgentImage)
	logLevel := s.getAgentSetting("agent.logLevel", defaultAgentLogLevel)
	portStr := s.getAgentSetting("agent.port", defaultAgentPort)
	natsURL := s.getAgentSetting("agent.nats.url", defaultNatsURL)
	redisURL := s.getAgentSetting("agent.redis.url", defaultRedisURL)
	qdrantURL := s.getAgentSetting("agent.qdrant.url", defaultQdrantURL)
	otelEndpoint := s.getAgentSetting("agent.otel.endpoint", defaultOtelEndpoint)
	s3Endpoint := s.getAgentSetting("agent.s3.endpoint", defaultS3Endpoint)
	s3Bucket := s.getAgentSetting("agent.s3.bucket", defaultS3Bucket)
	s3Region := s.getAgentSetting("agent.s3.region", defaultS3Region)
	caCertPath := s.getAgentSetting("agent.tls.caCertPath", defaultCaCertPath)

	port, err := strconv.Atoi(portStr)
	if err != nil {
		port = 8081
	}

	// 2. Map standard agent labels
	labels := map[string]string{
		"app.kubernetes.io/name":      "tacito-agent",
		"app.kubernetes.io/instance":  agent.Name,
		"app.kubernetes.io/part-of":   "tacito-square",
		"app.kubernetes.io/managed-by": "tacito-operator",
	}

	// 3. Resolve optional brain properties
	temp := "0.7"
	if agent.Spec.LLMConfig.Temperature != nil {
		temp = *agent.Spec.LLMConfig.Temperature
	}

	maxTokens := "2048"
	if agent.Spec.LLMConfig.MaxTokens != nil {
		maxTokens = strconv.Itoa(int(*agent.Spec.LLMConfig.MaxTokens))
	}

	// 4. Construct container environment variables
	env := []corev1.EnvVar{
		{Name: "TENANT_ID", Value: agent.Spec.TenantID},
		{Name: "TS_AGENT_NAME", Value: agent.Spec.AgentName},
		{Name: "TS_AGENT_COMMUNITY_REF", Value: agent.Spec.CommunityRef},
		{Name: "TS_AGENT_LOG_LEVEL", Value: logLevel},
		{Name: "TS_AGENT_PORT", Value: portStr},
		{Name: "TS_AGENT_NATS_URL", Value: natsURL},
		{Name: "TS_AGENT_REDIS_URL", Value: redisURL},
		{Name: "TS_AGENT_QDRANT_URL", Value: qdrantURL},
		{Name: "TS_AGENT_OTEL_ENDPOINT", Value: otelEndpoint},
		{Name: "TS_AGENT_S3_ENDPOINT", Value: s3Endpoint},
		{Name: "TS_AGENT_S3_BUCKET", Value: s3Bucket},
		{Name: "TS_AGENT_S3_REGION", Value: s3Region},
		{Name: "TS_AGENT_TLS_CA_CERT_PATH", Value: caCertPath},
		{Name: "TS_AGENT_BRAIN_MODEL", Value: agent.Spec.LLMConfig.Model},
		{Name: "TS_AGENT_BRAIN_TEMPERATURE", Value: temp},
		{Name: "TS_AGENT_BRAIN_MAX_TOKENS", Value: maxTokens},
	}

	if agent.Spec.SystemPrompt != "" {
		env = append(env, corev1.EnvVar{Name: "TS_AGENT_SYSTEM_PROMPT", Value: agent.Spec.SystemPrompt})
	}

	// 5. Construct container resource constraints
	var resources corev1.ResourceRequirements
	if agent.Spec.Resources != nil {
		resources = *agent.Spec.Resources
	}

	// 6. Resolve replica counts
	replicas := int32(1)
	if agent.Spec.Replicas != nil {
		replicas = *agent.Spec.Replicas
	}

	// 7. Generate Deployment manifest
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:            agent.Name,
			Namespace:       agent.Namespace,
			Labels:          labels,
			OwnerReferences: []metav1.OwnerReference{s.buildOwnerReference(agent)},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name":     "tacito-agent",
					"app.kubernetes.io/instance": agent.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/name":     "tacito-agent",
						"app.kubernetes.io/instance": agent.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "agent",
							Image:           image,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: int32(port),
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Env:       env,
							Resources: resources,
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/healthz",
										Port: intstr.FromString("http"),
									},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:        10,
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/readyz",
										Port: intstr.FromString("http"),
									},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:        10,
							},
						},
					},
				},
			},
		},
	}

	return deployment, nil
}

// BuildHeadlessService constructs the headless cluster Service specification for the agent.
func (s *ReconcileAgentServiceImpl) BuildHeadlessService(ctx context.Context, agent *v1alpha1.TacitoAgent) (*corev1.Service, error) {
	s.logger.Debug().Str("namespace", agent.Namespace).Str("name", agent.Name).Msg("entering BuildHeadlessService")
	portStr := s.getAgentSetting("agent.port", defaultAgentPort)
	port, err := strconv.Atoi(portStr)
	if err != nil {
		port = 8081
	}

	labels := map[string]string{
		"app.kubernetes.io/name":      "tacito-agent",
		"app.kubernetes.io/instance":  agent.Name,
		"app.kubernetes.io/part-of":   "tacito-square",
		"app.kubernetes.io/managed-by": "tacito-operator",
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            agent.Name,
			Namespace:       agent.Namespace,
			Labels:          labels,
			OwnerReferences: []metav1.OwnerReference{s.buildOwnerReference(agent)},
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: corev1.ClusterIPNone,
			Selector: map[string]string{
				"app.kubernetes.io/name":     "tacito-agent",
				"app.kubernetes.io/instance": agent.Name,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       int32(port),
					TargetPort: intstr.FromInt32(int32(port)),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}

	return service, nil
}

func (s *ReconcileAgentServiceImpl) getAgentSetting(key, defaultValue string) string {
	if s.cfg != nil && s.cfg.IsSet(key) {
		return s.cfg.GetString(key)
	}
	return defaultValue
}

func (s *ReconcileAgentServiceImpl) buildOwnerReference(agent *v1alpha1.TacitoAgent) metav1.OwnerReference {
	apiVersion := agent.APIVersion
	if apiVersion == "" {
		apiVersion = fmt.Sprintf("%s/%s", v1alpha1.GroupVersion.Group, v1alpha1.GroupVersion.Version)
	}
	kind := agent.Kind
	if kind == "" {
		kind = "TacitoAgent"
	}
	return metav1.OwnerReference{
		APIVersion:         apiVersion,
		Kind:               kind,
		Name:               agent.Name,
		UID:                agent.UID,
		Controller:         boolPtr(true),
		BlockOwnerDeletion: boolPtr(true),
	}
}

func boolPtr(b bool) *bool {
	return &b
}

