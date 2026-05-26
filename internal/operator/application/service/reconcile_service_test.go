package service_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/morphy76/tacito-square/internal/operator/application/service"
	"github.com/morphy76/tacito-square/pkg/kubernetes/apis/tacito/v1alpha1"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestBuildDeployment_TenantIDAndBasicEnv(t *testing.T) {
	logger := zerolog.Nop()
	cfg := viper.New()
	svc := service.NewReconcileAgentService(logger, cfg)

	agent := &v1alpha1.TacitoAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
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
	assert.Equal(t, "test-agent", dep.Name)
	assert.Equal(t, "tacito", dep.Namespace)

	// Assert labels
	assert.Equal(t, "tacito-agent", dep.Labels["app.kubernetes.io/name"])
	assert.Equal(t, "test-agent", dep.Labels["app.kubernetes.io/instance"])

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
	assert.Equal(t, "my-brainy-agent", envMap["TS_AGENT_NAME"])
	assert.Equal(t, "community-xyz", envMap["TS_AGENT_COMMUNITY_REF"])
	assert.Equal(t, "gpt-4o", envMap["TS_AGENT_BRAIN_MODEL"])
}

func TestBuildDeployment_LLMDefaultsAndOverrides(t *testing.T) {
	logger := zerolog.Nop()
	cfg := viper.New()
	svc := service.NewReconcileAgentService(logger, cfg)

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

	// Test 2: Overrides
	tempVal := "0.3"
	maxTokensVal := int32(4096)
	agentOverrides := &v1alpha1.TacitoAgent{
		Spec: v1alpha1.TacitoAgentSpec{
			LLMConfig: v1alpha1.LLMConfig{
				Model:       "gpt-4o",
				Temperature: &tempVal,
				MaxTokens:   &maxTokensVal,
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
}

func TestBuildDeployment_ResourceConstraints(t *testing.T) {
	logger := zerolog.Nop()
	cfg := viper.New()
	svc := service.NewReconcileAgentService(logger, cfg)

	cpuRequest := resource.MustParse("100m")
	memLimit := resource.MustParse("256Mi")

	agent := &v1alpha1.TacitoAgent{
		Spec: v1alpha1.TacitoAgentSpec{
			Resources: &corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU: cpuRequest,
				},
				Limits: corev1.ResourceList{
					corev1.ResourceMemory: memLimit,
				},
			},
		},
	}

	dep, err := svc.BuildDeployment(context.Background(), agent)
	assert.NoError(t, err)
	container := dep.Spec.Template.Spec.Containers[0]

	assert.Equal(t, cpuRequest, container.Resources.Requests[corev1.ResourceCPU])
	assert.Equal(t, memLimit, container.Resources.Limits[corev1.ResourceMemory])
}

func TestBuildDeployment_OwnerReference(t *testing.T) {
	logger := zerolog.Nop()
	cfg := viper.New()
	svc := service.NewReconcileAgentService(logger, cfg)

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
	svc := service.NewReconcileAgentService(logger, cfg)

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
