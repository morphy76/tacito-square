package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/morphy76/tacito-square/internal/operator/application/ports/inbound"
	"github.com/morphy76/tacito-square/pkg/kubernetes/apis/tacito/v1alpha1"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
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

// TierImage defines image configuration inside a tier profile.
type TierImage struct {
	Registry   string `yaml:"registry"`
	Name       string `yaml:"name"`
	Tag        string `yaml:"tag"`
	PullPolicy string `yaml:"pullPolicy"`
}

// TierResources defines CPU/memory limits & requests in a tier profile.
type TierResources struct {
	Requests map[string]string `yaml:"requests"`
	Limits   map[string]string `yaml:"limits"`
}

// ProbeTiming defines probe delay and period in a tier profile.
type ProbeTiming struct {
	InitialDelaySeconds int32 `yaml:"initialDelaySeconds"`
	PeriodSeconds       int32 `yaml:"periodSeconds"`
}

// TierProfile defines a logical runtime tier's spec overrides.
type TierProfile struct {
	Image          TierImage     `yaml:"image"`
	Resources      TierResources `yaml:"resources"`
	LivenessProbe  ProbeTiming   `yaml:"livenessProbe"`
	ReadinessProbe ProbeTiming   `yaml:"readinessProbe"`
}

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

	// 1. Resolve configurations with tier-based mapping or implicit defaults
	var tierProfile TierProfile
	var isTierResolved bool

	if agent.Spec.Tier != "" {
		if profile, exists := s.tierMap[agent.Spec.Tier]; exists {
			tierProfile = profile
			isTierResolved = true
			s.logger.Info().Str("tier", agent.Spec.Tier).Msg("resolved agent tier configuration profile")
		} else {
			s.logger.Warn().Str("tier", agent.Spec.Tier).Msg("requested tier not found in configuration, falling back to implicit default")
		}
	}

	// Resolve image & pullPolicy
	var image string
	var imagePullPolicy corev1.PullPolicy = corev1.PullIfNotPresent

	if isTierResolved && tierProfile.Image.Name != "" {
		registry := tierProfile.Image.Registry
		name := tierProfile.Image.Name
		tag := tierProfile.Image.Tag
		if registry != "" {
			image = fmt.Sprintf("%s/%s:%s", registry, name, tag)
		} else {
			image = fmt.Sprintf("%s:%s", name, tag)
		}

		if tierProfile.Image.PullPolicy != "" {
			imagePullPolicy = corev1.PullPolicy(tierProfile.Image.PullPolicy)
		}
	} else {
		image = s.getAgentSetting("agent.image", defaultAgentImage)
		if pullPolicyStr := s.getAgentSetting("agent.image.pullPolicy", ""); pullPolicyStr != "" {
			imagePullPolicy = corev1.PullPolicy(pullPolicyStr)
		}
	}

	// Resolve resources
	var resources corev1.ResourceRequirements
	if isTierResolved && (len(tierProfile.Resources.Requests) > 0 || len(tierProfile.Resources.Limits) > 0) {
		resources.Requests = make(corev1.ResourceList)
		for k, v := range tierProfile.Resources.Requests {
			resName := corev1.ResourceName(k)
			qty, err := resource.ParseQuantity(v)
			if err == nil {
				resources.Requests[resName] = qty
			}
		}
		resources.Limits = make(corev1.ResourceList)
		for k, v := range tierProfile.Resources.Limits {
			resName := corev1.ResourceName(k)
			qty, err := resource.ParseQuantity(v)
			if err == nil {
				resources.Limits[resName] = qty
			}
		}
	} else {
		// Build implicit default resources from Viper agent.resources.* values
		resources.Requests = make(corev1.ResourceList)
		resources.Limits = make(corev1.ResourceList)

		if reqCPU := s.getAgentSetting("agent.resources.requests.cpu", ""); reqCPU != "" {
			if qty, err := resource.ParseQuantity(reqCPU); err == nil {
				resources.Requests[corev1.ResourceCPU] = qty
			}
		}
		if reqMem := s.getAgentSetting("agent.resources.requests.memory", ""); reqMem != "" {
			if qty, err := resource.ParseQuantity(reqMem); err == nil {
				resources.Requests[corev1.ResourceMemory] = qty
			}
		}
		if limCPU := s.getAgentSetting("agent.resources.limits.cpu", ""); limCPU != "" {
			if qty, err := resource.ParseQuantity(limCPU); err == nil {
				resources.Limits[corev1.ResourceCPU] = qty
			}
		}
		if limMem := s.getAgentSetting("agent.resources.limits.memory", ""); limMem != "" {
			if qty, err := resource.ParseQuantity(limMem); err == nil {
				resources.Limits[corev1.ResourceMemory] = qty
			}
		}
	}

	// Resolve liveness & readiness probe timing overrides
	var livenessInitialDelay int32 = 1
	var livenessPeriod int32 = 1
	var readinessInitialDelay int32 = 1
	var readinessPeriod int32 = 1

	if isTierResolved && tierProfile.LivenessProbe.InitialDelaySeconds > 0 {
		livenessInitialDelay = tierProfile.LivenessProbe.InitialDelaySeconds
	}
	if isTierResolved && tierProfile.LivenessProbe.PeriodSeconds > 0 {
		livenessPeriod = tierProfile.LivenessProbe.PeriodSeconds
	}
	if isTierResolved && tierProfile.ReadinessProbe.InitialDelaySeconds > 0 {
		readinessInitialDelay = tierProfile.ReadinessProbe.InitialDelaySeconds
	}
	if isTierResolved && tierProfile.ReadinessProbe.PeriodSeconds > 0 {
		readinessPeriod = tierProfile.ReadinessProbe.PeriodSeconds
	}

	logLevel := s.getAgentSetting("agent.log.level", defaultAgentLogLevel)
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
		{Name: "TS_AGENT_BYPASS_LTM", Value: "true"},
	}

	if agent.Spec.LLMConfig.Endpoint != nil && *agent.Spec.LLMConfig.Endpoint != "" {
		env = append(env, corev1.EnvVar{
			Name:  "TS_AGENT_OPENAI_ENDPOINT",
			Value: *agent.Spec.LLMConfig.Endpoint,
		})
	}

	if agent.Spec.LLMConfig.CredentialsSecret != nil && *agent.Spec.LLMConfig.CredentialsSecret != "" {
		env = append(env, corev1.EnvVar{
			Name: "TS_AGENT_OPENAI_API_KEY",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: *agent.Spec.LLMConfig.CredentialsSecret,
					},
					Key: "api-key",
				},
			},
		})
	}

	if agent.Spec.SystemPrompt != "" {
		env = append(env, corev1.EnvVar{Name: "TS_AGENT_SYSTEM_PROMPT", Value: agent.Spec.SystemPrompt})
	}

	if len(agent.Spec.MCPClients) > 0 {
		mcpJSON, err := json.Marshal(agent.Spec.MCPClients)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal mcp clients to JSON: %w", err)
		}
		env = append(env, corev1.EnvVar{Name: "TS_AGENT_MCP_CLIENTS", Value: string(mcpJSON)})
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
					DNSPolicy: corev1.DNSClusterFirst,
					DNSConfig: &corev1.PodDNSConfig{
						Options: []corev1.PodDNSConfigOption{
							{
								Name:  "ndots",	
								Value: new("2"),
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:            "agent",
							Image:           image,
							ImagePullPolicy: imagePullPolicy,
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
								InitialDelaySeconds: livenessInitialDelay,
								PeriodSeconds:        livenessPeriod,
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/readyz",
										Port: intstr.FromString("http"),
									},
								},
								InitialDelaySeconds: readinessInitialDelay,
								PeriodSeconds:        readinessPeriod,
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

