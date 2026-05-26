package v1alpha1

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestLLMConfig_DeepCopy(t *testing.T) {
	temp := "0.7"
	tokens := int32(100)
	original := &LLMConfig{
		Model:       "gpt-4",
		Temperature: &temp,
		MaxTokens:   &tokens,
	}

	copied := original.DeepCopy()
	require.NotNil(t, copied)
	assert.Equal(t, original, copied)

	// Pointers must be distinct
	assert.NotSame(t, original, copied)
	assert.NotSame(t, original.Temperature, copied.Temperature)
	assert.NotSame(t, original.MaxTokens, copied.MaxTokens)

	// Modifying clone must not affect original
	newTemp := "0.9"
	*copied.Temperature = newTemp
	assert.Equal(t, "0.7", *original.Temperature)

	newTokens := int32(200)
	*copied.MaxTokens = newTokens
	assert.Equal(t, int32(100), *original.MaxTokens)

	// Nil pointers
	nilConfig := (*LLMConfig)(nil)
	assert.Nil(t, nilConfig.DeepCopy())
}

func TestTacitoAgentSpec_DeepCopy(t *testing.T) {
	replicas := int32(3)
	cpuLimit := resource.MustParse("500m")
	memLimit := resource.MustParse("512Mi")
	original := &TacitoAgentSpec{
		TenantID:     "t-1",
		AgentName:    "agent-1",
		CommunityRef: "comm-1",
		LLMConfig: LLMConfig{
			Model: "gpt-4",
		},
		SystemPrompt: "Hello",
		Replicas:     &replicas,
		Resources: &corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    cpuLimit,
				corev1.ResourceMemory: memLimit,
			},
		},
	}

	copied := original.DeepCopy()
	require.NotNil(t, copied)
	assert.Equal(t, original, copied)
	assert.NotSame(t, original, copied)
	assert.NotSame(t, original.Replicas, copied.Replicas)
	assert.NotSame(t, original.Resources, copied.Resources)

	// Modifying replicas in clone
	*copied.Replicas = 5
	assert.Equal(t, int32(3), *original.Replicas)

	// Modifying resources in clone
	newCpuLimit := resource.MustParse("1")
	copied.Resources.Limits[corev1.ResourceCPU] = newCpuLimit
	assert.Equal(t, "500m", original.Resources.Limits.Cpu().String())

	// Nil spec
	nilSpec := (*TacitoAgentSpec)(nil)
	assert.Nil(t, nilSpec.DeepCopy())
}

func TestTacitoAgentStatus_DeepCopy(t *testing.T) {
	now := metav1.NewTime(time.Now())
	original := &TacitoAgentStatus{
		Phase: PhaseRunning,
		Conditions: []metav1.Condition{
			{
				Type:   "Ready",
				Status: metav1.ConditionTrue,
			},
		},
		LastHeartbeat: now,
		Selector:      "app=tacito",
		Replicas:      3,
	}

	copied := original.DeepCopy()
	require.NotNil(t, copied)
	assert.Equal(t, original, copied)
	assert.NotSame(t, original, copied)
	assert.NotSame(t, &original.Conditions[0], &copied.Conditions[0])

	// Modifying conditions slice in clone
	copied.Conditions[0].Status = metav1.ConditionFalse
	assert.Equal(t, metav1.ConditionTrue, original.Conditions[0].Status)

	// Nil status
	nilStatus := (*TacitoAgentStatus)(nil)
	assert.Nil(t, nilStatus.DeepCopy())
}

func TestTacitoAgent_DeepCopy(t *testing.T) {
	replicas := int32(1)
	original := &TacitoAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "default",
			Labels:    map[string]string{"foo": "bar"},
		},
		Spec: TacitoAgentSpec{
			TenantID: "tenant-1",
			Replicas: &replicas,
		},
		Status: TacitoAgentStatus{
			Phase: PhasePending,
		},
	}

	copied := original.DeepCopy()
	require.NotNil(t, copied)
	assert.Equal(t, original, copied)
	assert.NotSame(t, original, copied)
	assert.NotSame(t, &original.Spec, &copied.Spec)
	assert.NotSame(t, &original.Status, &copied.Status)

	// Modifying ObjectMeta
	copied.Labels["foo"] = "baz"
	assert.Equal(t, "bar", original.Labels["foo"])

	// Nil agent
	nilAgent := (*TacitoAgent)(nil)
	assert.Nil(t, nilAgent.DeepCopy())
}

func TestTacitoAgentList_DeepCopy(t *testing.T) {
	original := &TacitoAgentList{
		Items: []TacitoAgent{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "agent-1",
				},
			},
		},
	}

	copied := original.DeepCopy()
	require.NotNil(t, copied)
	assert.Equal(t, original, copied)
	assert.NotSame(t, original, copied)
	assert.NotSame(t, &original.Items[0], &copied.Items[0])

	// Nil list
	nilList := (*TacitoAgentList)(nil)
	assert.Nil(t, nilList.DeepCopy())
}
