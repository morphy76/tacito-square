package v1alpha1

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// --- GroupVersion & Scheme Registration ---

func TestGroupVersion_IsCorrect(t *testing.T) {
	expected := schema.GroupVersion{Group: "tacito.square.io", Version: "v1alpha1"}
	assert.Equal(t, expected, GroupVersion)
}

func TestResource_ReturnsQualifiedGroupResource(t *testing.T) {
	gr := Resource("tacitoagents")
	assert.Equal(t, "tacitoagents", gr.Resource)
	assert.Equal(t, "tacito.square.io", gr.Group)
}

func TestAddToScheme_RegistersTypes(t *testing.T) {
	scheme := runtime.NewScheme()
	err := AddToScheme(scheme)
	require.NoError(t, err)

	gvk := GroupVersion.WithKind("TacitoAgent")
	obj, err := scheme.New(gvk)
	require.NoError(t, err)
	assert.IsType(t, &TacitoAgent{}, obj)
}

func TestAddToScheme_RegistersListType(t *testing.T) {
	scheme := runtime.NewScheme()
	err := AddToScheme(scheme)
	require.NoError(t, err)

	gvk := GroupVersion.WithKind("TacitoAgentList")
	obj, err := scheme.New(gvk)
	require.NoError(t, err)
	assert.IsType(t, &TacitoAgentList{}, obj)
}

// --- Phase Enum ---

func TestPhaseConstants_MatchExpectedValues(t *testing.T) {
	assert.Equal(t, TacitoAgentPhase("Pending"), PhasePending)
	assert.Equal(t, TacitoAgentPhase("Running"), PhaseRunning)
	assert.Equal(t, TacitoAgentPhase("Idle"), PhaseIdle)
	assert.Equal(t, TacitoAgentPhase("Terminated"), PhaseTerminated)
}

// --- JSON Serialization / Deserialization ---

func TestTacitoAgentSpec_JSONRoundTrip_AllFields(t *testing.T) {
	replicas := int32(3)
	temperature := "1.2"
	maxTokens := int32(4096)

	original := TacitoAgentSpec{
		TenantID:     "tenant-abc",
		AgentName:    "my-agent",
		CommunityRef: "community-xyz",
		LLMConfig: LLMConfig{
			Model:       "gpt-4o",
			Temperature: &temperature,
			MaxTokens:   &maxTokens,
		},
		SystemPrompt: "You are a helpful assistant.",
		Replicas:     &replicas,
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded TacitoAgentSpec
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original.TenantID, decoded.TenantID)
	assert.Equal(t, original.AgentName, decoded.AgentName)
	assert.Equal(t, original.CommunityRef, decoded.CommunityRef)
	assert.Equal(t, original.LLMConfig.Model, decoded.LLMConfig.Model)
	assert.Equal(t, *original.LLMConfig.Temperature, *decoded.LLMConfig.Temperature)
	assert.Equal(t, *original.LLMConfig.MaxTokens, *decoded.LLMConfig.MaxTokens)
	assert.Equal(t, original.SystemPrompt, decoded.SystemPrompt)
	assert.Equal(t, *original.Replicas, *decoded.Replicas)
}

func TestTacitoAgentSpec_JSONTags_RequiredFields(t *testing.T) {
	spec := TacitoAgentSpec{
		TenantID:     "t1",
		AgentName:    "a1",
		CommunityRef: "c1",
		LLMConfig: LLMConfig{
			Model: "gpt-4o",
		},
	}

	data, err := json.Marshal(spec)
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	// Required fields must be present in JSON with camelCase keys.
	assert.Contains(t, raw, "tenantId")
	assert.Contains(t, raw, "agentName")
	assert.Contains(t, raw, "communityRef")
	assert.Contains(t, raw, "llmConfig")

	llmConfig, ok := raw["llmConfig"].(map[string]interface{})
	require.True(t, ok, "llmConfig should be a JSON object")
	assert.Contains(t, llmConfig, "model")
}

func TestTacitoAgentSpec_JSONTags_OptionalFieldsOmitted(t *testing.T) {
	spec := TacitoAgentSpec{
		TenantID:     "t1",
		AgentName:    "a1",
		CommunityRef: "c1",
		LLMConfig: LLMConfig{
			Model: "gpt-4o",
		},
	}

	data, err := json.Marshal(spec)
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	// Optional fields with omitempty should be absent when zero/nil.
	assert.NotContains(t, raw, "systemPrompt")
	assert.NotContains(t, raw, "replicas")
	assert.NotContains(t, raw, "resources")

	llmConfig, ok := raw["llmConfig"].(map[string]interface{})
	require.True(t, ok)
	assert.NotContains(t, llmConfig, "temperature")
	assert.NotContains(t, llmConfig, "maxTokens")
}

func TestTacitoAgentSpec_DefaultNilForOptionalPointers(t *testing.T) {
	var spec TacitoAgentSpec
	assert.Nil(t, spec.Replicas)
	assert.Nil(t, spec.Resources)
	assert.Nil(t, spec.LLMConfig.Temperature)
	assert.Nil(t, spec.LLMConfig.MaxTokens)
}

func TestTacitoAgentSpec_JSONDeserialization_FromPayload(t *testing.T) {
	payload := `{
		"tenantId": "tenant-001",
		"agentName": "researcher",
		"communityRef": "science-team",
		"llmConfig": {
			"model": "gemini-1.5-pro",
			"temperature": "0.5",
			"maxTokens": 1024
		},
		"systemPrompt": "Analyze papers.",
		"replicas": 2
	}`

	var spec TacitoAgentSpec
	err := json.Unmarshal([]byte(payload), &spec)
	require.NoError(t, err)

	assert.Equal(t, "tenant-001", spec.TenantID)
	assert.Equal(t, "researcher", spec.AgentName)
	assert.Equal(t, "science-team", spec.CommunityRef)
	assert.Equal(t, "gemini-1.5-pro", spec.LLMConfig.Model)
	require.NotNil(t, spec.LLMConfig.Temperature)
	assert.Equal(t, "0.5", *spec.LLMConfig.Temperature)
	require.NotNil(t, spec.LLMConfig.MaxTokens)
	assert.Equal(t, int32(1024), *spec.LLMConfig.MaxTokens)
	assert.Equal(t, "Analyze papers.", spec.SystemPrompt)
	require.NotNil(t, spec.Replicas)
	assert.Equal(t, int32(2), *spec.Replicas)
}

// --- Status Fields ---

func TestTacitoAgentStatus_JSONRoundTrip(t *testing.T) {
	now := metav1.Now()
	status := TacitoAgentStatus{
		Phase: PhaseRunning,
		Conditions: []metav1.Condition{
			{
				Type:               "Available",
				Status:             metav1.ConditionTrue,
				LastTransitionTime: now,
				Reason:             "PodReady",
				Message:            "Agent pod is ready",
			},
		},
		LastHeartbeat: now,
		Replicas:      2,
		Selector:      "app=tacitoagent",
	}

	data, err := json.Marshal(status)
	require.NoError(t, err)

	var decoded TacitoAgentStatus
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, PhaseRunning, decoded.Phase)
	require.Len(t, decoded.Conditions, 1)
	assert.Equal(t, "Available", decoded.Conditions[0].Type)
	assert.Equal(t, int32(2), decoded.Replicas)
	assert.Equal(t, "app=tacitoagent", decoded.Selector)
}

// --- Scale Subresource Fields ---

func TestTacitoAgent_ScaleFields_ExistAndBind(t *testing.T) {
	replicas := int32(5)
	agent := TacitoAgent{
		Spec: TacitoAgentSpec{
			Replicas: &replicas,
		},
		Status: TacitoAgentStatus{
			Replicas: 3,
			Selector: "app=tacitoagent",
		},
	}

	assert.Equal(t, int32(5), *agent.Spec.Replicas)
	assert.Equal(t, int32(3), agent.Status.Replicas)
	assert.Equal(t, "app=tacitoagent", agent.Status.Selector)
}

// --- Full TacitoAgent JSON ---

func TestTacitoAgent_FullJSON_Roundtrip(t *testing.T) {
	replicas := int32(1)
	temperature := "0.7"
	maxTokens := int32(2048)

	agent := TacitoAgent{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "tacito.square.io/v1alpha1",
			Kind:       "TacitoAgent",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "default",
		},
		Spec: TacitoAgentSpec{
			TenantID:     "tenant-1",
			AgentName:    "test-agent",
			CommunityRef: "my-community",
			LLMConfig: LLMConfig{
				Model:       "gpt-4o",
				Temperature: &temperature,
				MaxTokens:   &maxTokens,
			},
			SystemPrompt: "You are a test agent.",
			Replicas:     &replicas,
		},
		Status: TacitoAgentStatus{
			Phase:    PhasePending,
			Replicas: 0,
		},
	}

	data, err := json.Marshal(agent)
	require.NoError(t, err)

	var decoded TacitoAgent
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "tacito.square.io/v1alpha1", decoded.APIVersion)
	assert.Equal(t, "TacitoAgent", decoded.Kind)
	assert.Equal(t, "test-agent", decoded.Name)
	assert.Equal(t, "tenant-1", decoded.Spec.TenantID)
	assert.Equal(t, PhasePending, decoded.Status.Phase)
}
