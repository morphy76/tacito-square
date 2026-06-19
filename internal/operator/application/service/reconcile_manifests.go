package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/morphy76/tacito-square/pkg/kubernetes/apis/tacito/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
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

	agentID := strings.TrimPrefix(agent.Name, "u-")

	role := agent.Spec.Role
	if role == "" {
		role = "spoke"
	}

	// 4. Construct container environment variables
	env := []corev1.EnvVar{
		{Name: "TENANT_ID", Value: agent.Spec.TenantID},
		{Name: "TS_AGENT_ID", Value: agentID},
		{Name: "TS_AGENT_NAME", Value: agent.Spec.AgentName},
		{Name: "TS_AGENT_COMMUNITY_REF", Value: agent.Spec.CommunityRef},
		{Name: "TS_AGENT_ROLE", Value: role},
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
								Value: newString("2"),
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

func newString(s string) *string {
	return &s
}
