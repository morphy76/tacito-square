package service_test

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"

	"github.com/morphy76/tacito-square/internal/operator/application/service"
	"github.com/morphy76/tacito-square/pkg/kubernetes/apis/tacito/v1alpha1"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestBuildDeployment_TenantIDAndBasicEnv(t *testing.T) {
	logger := zerolog.Nop()
	cfg := viper.New()
	fakeClient := fake.NewClientBuilder().Build()
	svc := service.NewReconcileAgentService(fakeClient, logger, cfg, nil)

	agent := &v1alpha1.TacitoAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "u-test-agent",
			Namespace: "tacito",
			UID:       types.UID("agent-uid-123"),
		},
		Spec: v1alpha1.TacitoAgentSpec{
			TenantID:     "tenant-abc",
			AgentName:    "my-brainy-agent",
			CommunityRef: "community-xyz",
			LLMConfig: v1alpha1.LLMConfig{
				Model: "gpt-4o",
			},
		},
	}

	dep, err := svc.BuildDeployment(context.Background(), agent)
	assert.NoError(t, err)
	assert.NotNil(t, dep)

	// Assert basic metadata
	assert.Equal(t, "u-test-agent", dep.Name)
	assert.Equal(t, "tacito", dep.Namespace)

	// Assert labels
	assert.Equal(t, "tacito-agent", dep.Labels["app.kubernetes.io/name"])
	assert.Equal(t, "u-test-agent", dep.Labels["app.kubernetes.io/instance"])

	// Assert container specs
	require := assert.New(t)
	require.Len(dep.Spec.Template.Spec.Containers, 1)
	container := dep.Spec.Template.Spec.Containers[0]

	// Find and verify environment variables
	envMap := make(map[string]string)
	for _, env := range container.Env {
		envMap[env.Name] = env.Value
	}

	assert.Equal(t, "tenant-abc", envMap["TENANT_ID"])
	assert.Equal(t, "test-agent", envMap["TS_AGENT_ID"])
	assert.Equal(t, "my-brainy-agent", envMap["TS_AGENT_NAME"])
	assert.Equal(t, "community-xyz", envMap["TS_AGENT_COMMUNITY_REF"])
	assert.Equal(t, "gpt-4o", envMap["TS_AGENT_BRAIN_MODEL"])
	assert.Equal(t, "spoke", envMap["TS_AGENT_ROLE"])
}

func TestBuildDeployment_LLMDefaultsAndOverrides(t *testing.T) {
	logger := zerolog.Nop()
	cfg := viper.New()
	fakeClient := fake.NewClientBuilder().Build()
	svc := service.NewReconcileAgentService(fakeClient, logger, cfg, nil)

	// Test 1: Defaults fallback
	agentDefaults := &v1alpha1.TacitoAgent{
		Spec: v1alpha1.TacitoAgentSpec{
			LLMConfig: v1alpha1.LLMConfig{
				Model: "gpt-4o",
			},
		},
	}

	dep, err := svc.BuildDeployment(context.Background(), agentDefaults)
	assert.NoError(t, err)
	envMap := make(map[string]string)
	for _, env := range dep.Spec.Template.Spec.Containers[0].Env {
		envMap[env.Name] = env.Value
	}

	assert.Equal(t, "0.7", envMap["TS_AGENT_BRAIN_TEMPERATURE"])
	assert.Equal(t, "2048", envMap["TS_AGENT_BRAIN_MAX_TOKENS"])
	_, hasEndpoint := envMap["TS_AGENT_OPENAI_ENDPOINT"]
	assert.False(t, hasEndpoint)
	_, hasAPIKey := envMap["TS_AGENT_OPENAI_API_KEY"]
	assert.False(t, hasAPIKey)

	// Test 2: Overrides
	tempVal := "0.3"
	maxTokensVal := int32(4096)
	endpointVal := "https://api.openai.com/v1"
	secretVal := "my-openai-key-secret"
	agentOverrides := &v1alpha1.TacitoAgent{
		Spec: v1alpha1.TacitoAgentSpec{
			LLMConfig: v1alpha1.LLMConfig{
				Model:             "gpt-4o",
				Temperature:       &tempVal,
				MaxTokens:         &maxTokensVal,
				Endpoint:          &endpointVal,
				CredentialsSecret: &secretVal,
			},
			SystemPrompt: "Be extremely friendly.",
		},
	}

	depOverrides, err := svc.BuildDeployment(context.Background(), agentOverrides)
	assert.NoError(t, err)
	envMapOverrides := make(map[string]string)
	for _, env := range depOverrides.Spec.Template.Spec.Containers[0].Env {
		envMapOverrides[env.Name] = env.Value
	}

	assert.Equal(t, "0.3", envMapOverrides["TS_AGENT_BRAIN_TEMPERATURE"])
	assert.Equal(t, "4096", envMapOverrides["TS_AGENT_BRAIN_MAX_TOKENS"])
	assert.Equal(t, "Be extremely friendly.", envMapOverrides["TS_AGENT_SYSTEM_PROMPT"])
	assert.Equal(t, "https://api.openai.com/v1", envMapOverrides["TS_AGENT_OPENAI_ENDPOINT"])

	var foundAPIKey bool
	for _, env := range depOverrides.Spec.Template.Spec.Containers[0].Env {
		if env.Name == "TS_AGENT_OPENAI_API_KEY" {
			foundAPIKey = true
			assert.Empty(t, env.Value)
			assert.NotNil(t, env.ValueFrom)
			assert.NotNil(t, env.ValueFrom.SecretKeyRef)
			assert.Equal(t, "my-openai-key-secret", env.ValueFrom.SecretKeyRef.Name)
			assert.Equal(t, "api-key", env.ValueFrom.SecretKeyRef.Key)
		}
	}
	assert.True(t, foundAPIKey, "TS_AGENT_OPENAI_API_KEY environment variable was not set")
}

func TestBuildDeployment_OwnerReference(t *testing.T) {
	logger := zerolog.Nop()
	cfg := viper.New()
	fakeClient := fake.NewClientBuilder().Build()
	svc := service.NewReconcileAgentService(fakeClient, logger, cfg, nil)

	agent := &v1alpha1.TacitoAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "tacito",
			UID:       types.UID("agent-uid-123"),
		},
	}

	dep, err := svc.BuildDeployment(context.Background(), agent)
	assert.NoError(t, err)

	assert.Len(t, dep.OwnerReferences, 1)
	ownerRef := dep.OwnerReferences[0]
	assert.Equal(t, "test-agent", ownerRef.Name)
	assert.Equal(t, types.UID("agent-uid-123"), ownerRef.UID)
	assert.True(t, *ownerRef.Controller)
	assert.True(t, *ownerRef.BlockOwnerDeletion)
}

func TestBuildHeadlessService(t *testing.T) {
	logger := zerolog.Nop()
	cfg := viper.New()
	fakeClient := fake.NewClientBuilder().Build()
	svc := service.NewReconcileAgentService(fakeClient, logger, cfg, nil)

	agent := &v1alpha1.TacitoAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "tacito",
			UID:       types.UID("agent-uid-123"),
		},
	}

	svcObj, err := svc.BuildHeadlessService(context.Background(), agent)
	assert.NoError(t, err)
	assert.NotNil(t, svcObj)

	// Assert basic metadata
	assert.Equal(t, "test-agent", svcObj.Name)
	assert.Equal(t, "tacito", svcObj.Namespace)

	// Assert headless Service properties
	assert.Equal(t, corev1.ClusterIPNone, svcObj.Spec.ClusterIP)

	// Assert label selectors
	assert.Equal(t, "tacito-agent", svcObj.Spec.Selector["app.kubernetes.io/name"])
	assert.Equal(t, "test-agent", svcObj.Spec.Selector["app.kubernetes.io/instance"])

	// Assert ports mapping
	assert.Len(t, svcObj.Spec.Ports, 1)
	port := svcObj.Spec.Ports[0]
	assert.Equal(t, int32(8081), port.Port)
	assert.Equal(t, "http", port.Name)
	assert.Equal(t, strconv.Itoa(8081), port.TargetPort.String())

	// Assert OwnerReference
	assert.Len(t, svcObj.OwnerReferences, 1)
	ownerRef := svcObj.OwnerReferences[0]
	assert.Equal(t, "test-agent", ownerRef.Name)
	assert.Equal(t, types.UID("agent-uid-123"), ownerRef.UID)
	assert.True(t, *ownerRef.Controller)
}

func TestReconcile_CreatesDeploymentAndService(t *testing.T) {
	logger := zerolog.Nop()
	cfg := viper.New()

	scheme := runtime.NewScheme()
	err := v1alpha1.AddToScheme(scheme)
	require.NoError(t, err)
	err = appsv1.AddToScheme(scheme)
	require.NoError(t, err)
	err = corev1.AddToScheme(scheme)
	require.NoError(t, err)

	agent := &v1alpha1.TacitoAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-test-agent",
			Namespace: "tacito-ns",
			UID:       types.UID("uid-val-456"),
		},
		Spec: v1alpha1.TacitoAgentSpec{
			TenantID:     "tenant-1",
			AgentName:    "agent-1",
			CommunityRef: "comm-1",
			LLMConfig: v1alpha1.LLMConfig{
				Model: "gpt-4",
			},
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&v1alpha1.TacitoAgent{}).WithObjects(agent).Build()
	svc := service.NewReconcileAgentService(fakeClient, logger, cfg, nil)

	err = svc.Reconcile(context.Background(), agent)
	assert.NoError(t, err)

	// In TDD RED phase, the Reconcile method is a stub and does not create these K8s resources, so this test WILL fail (RED).
	var dep appsv1.Deployment
	err = fakeClient.Get(context.Background(), client.ObjectKey{Namespace: "tacito-ns", Name: "my-test-agent"}, &dep)
	assert.NoError(t, err, "Expected Deployment to be created in fake client")

	var serviceObj corev1.Service
	err = fakeClient.Get(context.Background(), client.ObjectKey{Namespace: "tacito-ns", Name: "my-test-agent"}, &serviceObj)
	assert.NoError(t, err, "Expected Service to be created in fake client")
}

func TestReconcile_StatusTransitions(t *testing.T) {
	logger := zerolog.Nop()
	cfg := viper.New()

	scheme := runtime.NewScheme()
	err := v1alpha1.AddToScheme(scheme)
	require.NoError(t, err)
	err = appsv1.AddToScheme(scheme)
	require.NoError(t, err)
	err = corev1.AddToScheme(scheme)
	require.NoError(t, err)

	agent := &v1alpha1.TacitoAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "status-agent",
			Namespace: "tacito-ns",
			UID:       types.UID("uid-val-789"),
		},
		Spec: v1alpha1.TacitoAgentSpec{
			TenantID:     "tenant-1",
			AgentName:    "agent-1",
			CommunityRef: "comm-1",
			LLMConfig: v1alpha1.LLMConfig{
				Model: "gpt-4",
			},
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&v1alpha1.TacitoAgent{}).WithObjects(agent).Build()
	svc := service.NewReconcileAgentService(fakeClient, logger, cfg, nil)

	// 1. Initial Reconcile -> Pending (0 ready replicas)
	err = svc.Reconcile(context.Background(), agent)
	assert.NoError(t, err)

	// Fetch updated agent from fake client to verify status
	var updatedAgent v1alpha1.TacitoAgent
	err = fakeClient.Get(context.Background(), client.ObjectKey{Namespace: "tacito-ns", Name: "status-agent"}, &updatedAgent)
	assert.NoError(t, err)
	assert.Equal(t, v1alpha1.PhasePending, updatedAgent.Status.Phase)
	assert.Equal(t, int32(0), updatedAgent.Status.Replicas)
	assert.Len(t, updatedAgent.Status.Conditions, 1)
	assert.Equal(t, "Available", updatedAgent.Status.Conditions[0].Type)
	assert.Equal(t, metav1.ConditionFalse, updatedAgent.Status.Conditions[0].Status)
	assert.Equal(t, "NoReplicasAvailable", updatedAgent.Status.Conditions[0].Reason)

	// 2. Deployment status updated to 1 Ready Replica -> Reconcile -> Running
	var dep appsv1.Deployment
	err = fakeClient.Get(context.Background(), client.ObjectKey{Namespace: "tacito-ns", Name: "status-agent"}, &dep)
	assert.NoError(t, err)
	dep.Status.ReadyReplicas = 1
	err = fakeClient.Status().Update(context.Background(), &dep)
	assert.NoError(t, err)

	err = svc.Reconcile(context.Background(), &updatedAgent)
	assert.NoError(t, err)

	err = fakeClient.Get(context.Background(), client.ObjectKey{Namespace: "tacito-ns", Name: "status-agent"}, &updatedAgent)
	assert.NoError(t, err)
	assert.Equal(t, v1alpha1.PhaseRunning, updatedAgent.Status.Phase)
	assert.Equal(t, int32(1), updatedAgent.Status.Replicas)
	assert.Equal(t, metav1.ConditionTrue, updatedAgent.Status.Conditions[0].Status)
	assert.Equal(t, "MinimumReplicasAvailable", updatedAgent.Status.Conditions[0].Reason)

	// 3. Scale agent spec to 0 replicas -> Reconcile -> Idle
	zero := int32(0)
	updatedAgent.Spec.Replicas = &zero
	err = fakeClient.Update(context.Background(), &updatedAgent)
	assert.NoError(t, err)

	// Also simulate the deployment getting updated to 0 replicas by operator and ready replicas dropping to 0
	err = fakeClient.Get(context.Background(), client.ObjectKey{Namespace: "tacito-ns", Name: "status-agent"}, &dep)
	assert.NoError(t, err)
	dep.Spec.Replicas = &zero
	err = fakeClient.Update(context.Background(), &dep)
	assert.NoError(t, err)

	dep.Status.ReadyReplicas = 0
	err = fakeClient.Status().Update(context.Background(), &dep)
	assert.NoError(t, err)

	err = svc.Reconcile(context.Background(), &updatedAgent)
	assert.NoError(t, err)

	err = fakeClient.Get(context.Background(), client.ObjectKey{Namespace: "tacito-ns", Name: "status-agent"}, &updatedAgent)
	assert.NoError(t, err)
	assert.Equal(t, v1alpha1.PhaseIdle, updatedAgent.Status.Phase)
	assert.Equal(t, int32(0), updatedAgent.Status.Replicas)
	assert.Equal(t, metav1.ConditionFalse, updatedAgent.Status.Conditions[0].Status)
	assert.Equal(t, "ScaleToZero", updatedAgent.Status.Conditions[0].Reason)
}

func TestBuildDeployment_WithMCPClients(t *testing.T) {
	logger := zerolog.Nop()
	cfg := viper.New()
	fakeClient := fake.NewClientBuilder().Build()
	svc := service.NewReconcileAgentService(fakeClient, logger, cfg, nil)

	agent := &v1alpha1.TacitoAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mcp-agent",
			Namespace: "tacito",
		},
		Spec: v1alpha1.TacitoAgentSpec{
			TenantID:     "tenant-1",
			AgentName:    "mcp-agent-1",
			CommunityRef: "comm-1",
			LLMConfig: v1alpha1.LLMConfig{
				Model: "gpt-4",
			},
			MCPClients: []v1alpha1.MCPClientSpec{
				{
					Name:         "sqlite-mcp",
					Transport:    "stdio",
					Command:      "npx",
					Args:         []string{"@modelcontextprotocol/server-sqlite", "--db", "test.db"},
					Env:          map[string]string{"DEBUG": "true"},
					AllowedTools: []string{"query_db", "get_schema"},
				},
			},
		},
	}

	dep, err := svc.BuildDeployment(context.Background(), agent)
	assert.NoError(t, err)
	require.NotNil(t, dep)

	container := dep.Spec.Template.Spec.Containers[0]
	envMap := make(map[string]string)
	for _, env := range container.Env {
		envMap[env.Name] = env.Value
	}

	val, exists := envMap["TS_AGENT_MCP_CLIENTS"]
	require.True(t, exists, "Expected TS_AGENT_MCP_CLIENTS environment variable to exist")

	var clients []v1alpha1.MCPClientSpec
	err = json.Unmarshal([]byte(val), &clients)
	require.NoError(t, err, "Expected TS_AGENT_MCP_CLIENTS value to be valid JSON")
	require.Len(t, clients, 1)

	clientSpec := clients[0]
	assert.Equal(t, "sqlite-mcp", clientSpec.Name)
	assert.Equal(t, "stdio", clientSpec.Transport)
	assert.Equal(t, "npx", clientSpec.Command)
	assert.Equal(t, []string{"@modelcontextprotocol/server-sqlite", "--db", "test.db"}, clientSpec.Args)
	assert.Equal(t, "true", clientSpec.Env["DEBUG"])
	assert.Equal(t, []string{"query_db", "get_schema"}, clientSpec.AllowedTools)
}

func TestBuildDeployment_TierResolution(t *testing.T) {
	logger := zerolog.Nop()
	cfg := viper.New()
	// Setup implicit default settings in viper
	cfg.Set("agent.image", "my-default-agent:latest")
	cfg.Set("agent.resources.requests.cpu", "100m")
	cfg.Set("agent.resources.limits.memory", "256Mi")

	tierMap := map[string]service.TierProfile{
		"heavy": {
			Image: service.TierImage{
				Name: "tacito-square/agent-heavy",
				Tag:  "v1.0.0",
			},
			Resources: service.TierResources{
				Requests: map[string]string{"cpu": "500m"},
				Limits:   map[string]string{"memory": "1Gi"},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().Build()
	svc := service.NewReconcileAgentService(fakeClient, logger, cfg, tierMap)

	// Test 1: Resolve known tier "heavy"
	agentHeavy := &v1alpha1.TacitoAgent{
		Spec: v1alpha1.TacitoAgentSpec{
			Tier: "heavy",
		},
	}
	depHeavy, err := svc.BuildDeployment(context.Background(), agentHeavy)
	assert.NoError(t, err)
	containerHeavy := depHeavy.Spec.Template.Spec.Containers[0]
	assert.Equal(t, "tacito-square/agent-heavy:v1.0.0", containerHeavy.Image)
	assert.Equal(t, "500m", containerHeavy.Resources.Requests.Cpu().String())
	assert.Equal(t, "1Gi", containerHeavy.Resources.Limits.Memory().String())

	// Test 2: Fallback on unknown/empty tier
	agentDefault := &v1alpha1.TacitoAgent{
		Spec: v1alpha1.TacitoAgentSpec{
			Tier: "non-existent",
		},
	}
	depDefault, err := svc.BuildDeployment(context.Background(), agentDefault)
	assert.NoError(t, err)
	containerDefault := depDefault.Spec.Template.Spec.Containers[0]
	assert.Equal(t, "my-default-agent:latest", containerDefault.Image)
	assert.Equal(t, "100m", containerDefault.Resources.Requests.Cpu().String())
	assert.Equal(t, "256Mi", containerDefault.Resources.Limits.Memory().String())
}

func TestBuildDeployment_RoleMapping(t *testing.T) {
	logger := zerolog.Nop()
	cfg := viper.New()
	fakeClient := fake.NewClientBuilder().Build()
	svc := service.NewReconcileAgentService(fakeClient, logger, cfg, nil)

	// Test 1: Explicit Role "hub"
	agentHub := &v1alpha1.TacitoAgent{
		Spec: v1alpha1.TacitoAgentSpec{
			Role: "hub",
		},
	}
	depHub, err := svc.BuildDeployment(context.Background(), agentHub)
	assert.NoError(t, err)
	envMapHub := make(map[string]string)
	for _, env := range depHub.Spec.Template.Spec.Containers[0].Env {
		envMapHub[env.Name] = env.Value
	}
	assert.Equal(t, "hub", envMapHub["TS_AGENT_ROLE"])

	// Test 2: Explicit Role "spoke"
	agentSpoke := &v1alpha1.TacitoAgent{
		Spec: v1alpha1.TacitoAgentSpec{
			Role: "spoke",
		},
	}
	depSpoke, err := svc.BuildDeployment(context.Background(), agentSpoke)
	assert.NoError(t, err)
	envMapSpoke := make(map[string]string)
	for _, env := range depSpoke.Spec.Template.Spec.Containers[0].Env {
		envMapSpoke[env.Name] = env.Value
	}
	assert.Equal(t, "spoke", envMapSpoke["TS_AGENT_ROLE"])
}
