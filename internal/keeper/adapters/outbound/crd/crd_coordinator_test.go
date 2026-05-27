package crd_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	crdadapter "github.com/morphy76/tacito-square/internal/keeper/adapters/outbound/crd"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
	"github.com/morphy76/tacito-square/pkg/kubernetes/apis/tacito/v1alpha1"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestSubmitAgentCRD_CreateSuccess(t *testing.T) {
	scheme := runtime.NewScheme()
	err := v1alpha1.AddToScheme(scheme)
	require.NoError(t, err)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	coordinator := crdadapter.NewK8sCRDCoordinatorWithClient(fakeClient, "tacito", nil)

	agentID := uuid.New()
	communityID := uuid.New()
	agent := &model.Agent{
		ID:          agentID,
		TenantID:    "tenant-1",
		Name:        "agent-1",
		Description: "A helpful assistant",
		Brain: model.BrainConfig{
			Model:       "gpt-4",
			Temperature: 0.5,
			MaxTokens:   1000,
		},
		CommunityID: &communityID,
	}

	err = coordinator.SubmitAgentCRD(context.Background(), agent)
	assert.NoError(t, err)

	// Fetch custom resource from fake client
	fetched := &v1alpha1.TacitoAgent{}
	key := types.NamespacedName{Namespace: "tacito", Name: agentID.String()}
	err = fakeClient.Get(context.Background(), key, fetched)
	assert.NoError(t, err)

	// Assert mapped values
	assert.Equal(t, "tenant-1", fetched.Spec.TenantID)
	assert.Equal(t, "agent-1", fetched.Spec.AgentName)
	assert.Equal(t, communityID.String(), fetched.Spec.CommunityRef)
	assert.Equal(t, "gpt-4", fetched.Spec.LLMConfig.Model)
	assert.Equal(t, "0.5", *fetched.Spec.LLMConfig.Temperature)
	assert.Equal(t, int32(1000), *fetched.Spec.LLMConfig.MaxTokens)
}

func TestSubmitAgentCRD_UpdateSuccess(t *testing.T) {
	scheme := runtime.NewScheme()
	err := v1alpha1.AddToScheme(scheme)
	require.NoError(t, err)

	agentID := uuid.New()
	communityID := uuid.New()

	// Pre-populate an existing CRD
	existing := &v1alpha1.TacitoAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      agentID.String(),
			Namespace: "tacito",
		},
		Spec: v1alpha1.TacitoAgentSpec{
			TenantID:     "tenant-1",
			AgentName:    "old-agent-name",
			CommunityRef: communityID.String(),
			LLMConfig: v1alpha1.LLMConfig{
				Model: "gpt-3.5",
			},
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existing).Build()
	coordinator := crdadapter.NewK8sCRDCoordinatorWithClient(fakeClient, "tacito", nil)

	// Mapped model updates name and model
	agent := &model.Agent{
		ID:          agentID,
		TenantID:    "tenant-1",
		Name:        "new-agent-name",
		Description: "A helpful assistant",
		Brain: model.BrainConfig{
			Model:       "gpt-4",
			Temperature: 0.7,
			MaxTokens:   2000,
		},
		CommunityID: &communityID,
	}

	err = coordinator.SubmitAgentCRD(context.Background(), agent)
	assert.NoError(t, err)

	// Fetch custom resource from fake client and check updates
	fetched := &v1alpha1.TacitoAgent{}
	key := types.NamespacedName{Namespace: "tacito", Name: agentID.String()}
	err = fakeClient.Get(context.Background(), key, fetched)
	assert.NoError(t, err)

	assert.Equal(t, "new-agent-name", fetched.Spec.AgentName)
	assert.Equal(t, "gpt-4", fetched.Spec.LLMConfig.Model)
	assert.Equal(t, "0.7", *fetched.Spec.LLMConfig.Temperature)
	assert.Equal(t, int32(2000), *fetched.Spec.LLMConfig.MaxTokens)
}

func TestSubmitAgentCRD_ConflictResolution(t *testing.T) {
	scheme := runtime.NewScheme()
	err := v1alpha1.AddToScheme(scheme)
	require.NoError(t, err)

	agentID := uuid.New()
	communityID := uuid.New()

	// Pre-populate an existing CRD
	existing := &v1alpha1.TacitoAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      agentID.String(),
			Namespace: "tacito",
		},
		Spec: v1alpha1.TacitoAgentSpec{
			TenantID:     "tenant-1",
			AgentName:    "agent-old",
			CommunityRef: communityID.String(),
		},
	}

	// Setup client interceptor to return Conflict on the first Update attempt
	updateAttempts := 0
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(existing).
		WithInterceptorFuncs(interceptor.Funcs{
			Update: func(ctx context.Context, cl client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
				updateAttempts++
				if updateAttempts == 1 {
					// Return conflict error on first update attempt
					return apierrors.NewConflict(schema.GroupResource{Group: "tacito.square.io", Resource: "tacitoagents"}, obj.GetName(), errors.New("optimistic lock conflict"))
				}
				return cl.Update(ctx, obj, opts...)
			},
		}).
		Build()

	coordinator := crdadapter.NewK8sCRDCoordinatorWithClient(fakeClient, "tacito", nil)

	agent := &model.Agent{
		ID:          agentID,
		TenantID:    "tenant-1",
		Name:        "agent-new",
		Description: "A helpful assistant",
		Brain: model.BrainConfig{
			Model:       "gpt-4",
			Temperature: 0.5,
			MaxTokens:   1000,
		},
		CommunityID: &communityID,
	}

	err = coordinator.SubmitAgentCRD(context.Background(), agent)
	assert.NoError(t, err)

	// Assert conflict update was retried and completed successfully
	assert.Equal(t, 2, updateAttempts)

	// Verify updates persisted
	fetched := &v1alpha1.TacitoAgent{}
	key := types.NamespacedName{Namespace: "tacito", Name: agentID.String()}
	err = fakeClient.Get(context.Background(), key, fetched)
	assert.NoError(t, err)
	assert.Equal(t, "agent-new", fetched.Spec.AgentName)
}

func TestSubmitAgentCRD_Timeout(t *testing.T) {
	scheme := runtime.NewScheme()
	err := v1alpha1.AddToScheme(scheme)
	require.NoError(t, err)

	agentID := uuid.New()

	// Setup client interceptor to delay Kube-API operations to exceed deadline
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithInterceptorFuncs(interceptor.Funcs{
			Get: func(ctx context.Context, cl client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				select {
				case <-time.After(10 * time.Millisecond):
					return apierrors.NewNotFound(schema.GroupResource{Group: "tacito.square.io", Resource: "tacitoagents"}, key.Name)
				case <-ctx.Done():
					return ctx.Err()
				}
			},
			Create: func(ctx context.Context, cl client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
				select {
				case <-time.After(10 * time.Millisecond):
					return cl.Create(ctx, obj, opts...)
				case <-ctx.Done():
					return ctx.Err()
				}
			},
		}).
		Build()

	coordinator := crdadapter.NewK8sCRDCoordinatorWithClient(fakeClient, "tacito", nil)

	agent := &model.Agent{
		ID:       agentID,
		TenantID: "tenant-1",
		Name:     "agent-1",
	}

	// Pass context that is cancelled immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = coordinator.SubmitAgentCRD(ctx, agent)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded))
}

func TestTeardownAgentCRD_Success(t *testing.T) {
	scheme := runtime.NewScheme()
	err := v1alpha1.AddToScheme(scheme)
	require.NoError(t, err)

	agentID := uuid.New()
	existing := &v1alpha1.TacitoAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      agentID.String(),
			Namespace: "tacito",
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existing).Build()
	coordinator := crdadapter.NewK8sCRDCoordinatorWithClient(fakeClient, "tacito", nil)

	agent := &model.Agent{
		ID: agentID,
	}

	// Delete existing
	err = coordinator.TeardownAgentCRD(context.Background(), agent)
	assert.NoError(t, err)

	// Assert deleted
	fetched := &v1alpha1.TacitoAgent{}
	key := types.NamespacedName{Namespace: "tacito", Name: agentID.String()}
	err = fakeClient.Get(context.Background(), key, fetched)
	assert.Error(t, err)
	assert.True(t, apierrors.IsNotFound(err))

	// Graceful NotFound handling — deleting a non-existent agent returns no error
	err = coordinator.TeardownAgentCRD(context.Background(), agent)
	assert.NoError(t, err)
}

func startTestNatsServer(t *testing.T) (*server.Server, *nats.Conn) {
	opts := &server.Options{
		Host: "127.0.0.1",
		Port: -1,
	}
	ns, err := server.NewServer(opts)
	require.NoError(t, err)

	go ns.Start()
	if !ns.ReadyForConnections(2 * time.Second) {
		t.Fatal("NATS server not ready for connections")
	}

	nc, err := nats.Connect(ns.ClientURL())
	require.NoError(t, err)

	return ns, nc
}

type ProvisioningEvent struct {
	TenantID    string `json:"tenant_id"`
	AgentID     string `json:"agent_id"`
	CommunityID string `json:"community_id"`
	Timestamp   string `json:"timestamp"`
	Error       string `json:"error,omitempty"`
}

func TestSubmitAgentCRD_NATSProgressionStarted(t *testing.T) {
	ns, nc := startTestNatsServer(t)
	defer ns.Shutdown()
	defer nc.Close()

	scheme := runtime.NewScheme()
	err := v1alpha1.AddToScheme(scheme)
	require.NoError(t, err)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	coordinator := crdadapter.NewK8sCRDCoordinatorWithClient(fakeClient, "tacito", nc)

	agentID := uuid.New()
	communityID := uuid.New()
	agent := &model.Agent{
		ID:          agentID,
		TenantID:    "tenant-1",
		Name:        "agent-1",
		CommunityID: &communityID,
	}

	// Subscribe to Started event
	subChan := make(chan *nats.Msg, 10)
	sub, err := nc.ChanSubscribe("agent.provisioning.started", subChan)
	require.NoError(t, err)
	defer sub.Unsubscribe()

	err = coordinator.SubmitAgentCRD(context.Background(), agent)
	assert.NoError(t, err)

	select {
	case msg := <-subChan:
		var event ProvisioningEvent
		err = json.Unmarshal(msg.Data, &event)
		require.NoError(t, err)
		assert.Equal(t, "tenant-1", event.TenantID)
		assert.Equal(t, agentID.String(), event.AgentID)
		assert.Equal(t, communityID.String(), event.CommunityID)
		assert.NotEmpty(t, event.Timestamp)
		assert.Empty(t, event.Error)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for agent.provisioning.started event")
	}
}

func TestSubmitAgentCRD_NATSProgressionCompleted(t *testing.T) {
	ns, nc := startTestNatsServer(t)
	defer ns.Shutdown()
	defer nc.Close()

	scheme := runtime.NewScheme()
	err := v1alpha1.AddToScheme(scheme)
	require.NoError(t, err)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	coordinator := crdadapter.NewK8sCRDCoordinatorWithClient(fakeClient, "tacito", nc)

	agentID := uuid.New()
	agent := &model.Agent{
		ID:       agentID,
		TenantID: "tenant-2",
		Name:     "agent-2",
	}

	// Subscribe to Completed event
	subChan := make(chan *nats.Msg, 10)
	sub, err := nc.ChanSubscribe("agent.provisioning.completed", subChan)
	require.NoError(t, err)
	defer sub.Unsubscribe()

	err = coordinator.SubmitAgentCRD(context.Background(), agent)
	assert.NoError(t, err)

	select {
	case msg := <-subChan:
		var event ProvisioningEvent
		err = json.Unmarshal(msg.Data, &event)
		require.NoError(t, err)
		assert.Equal(t, "tenant-2", event.TenantID)
		assert.Equal(t, agentID.String(), event.AgentID)
		assert.Empty(t, event.CommunityID)
		assert.NotEmpty(t, event.Timestamp)
		assert.Empty(t, event.Error)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for agent.provisioning.completed event")
	}
}

func TestSubmitAgentCRD_NATSProgressionFailed(t *testing.T) {
	ns, nc := startTestNatsServer(t)
	defer ns.Shutdown()
	defer nc.Close()

	scheme := runtime.NewScheme()
	err := v1alpha1.AddToScheme(scheme)
	require.NoError(t, err)

	// Inject error in Create operation to trigger failure
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithInterceptorFuncs(interceptor.Funcs{
			Create: func(ctx context.Context, cl client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
				return errors.New("simulated API server write failure")
			},
		}).
		Build()

	coordinator := crdadapter.NewK8sCRDCoordinatorWithClient(fakeClient, "tacito", nc)

	agentID := uuid.New()
	agent := &model.Agent{
		ID:       agentID,
		TenantID: "tenant-3",
		Name:     "agent-3",
	}

	// Subscribe to Failed event
	subChan := make(chan *nats.Msg, 10)
	sub, err := nc.ChanSubscribe("agent.provisioning.failed", subChan)
	require.NoError(t, err)
	defer sub.Unsubscribe()

	err = coordinator.SubmitAgentCRD(context.Background(), agent)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "simulated API server write failure")

	select {
	case msg := <-subChan:
		var event ProvisioningEvent
		err = json.Unmarshal(msg.Data, &event)
		require.NoError(t, err)
		assert.Equal(t, "tenant-3", event.TenantID)
		assert.Equal(t, agentID.String(), event.AgentID)
		assert.NotEmpty(t, event.Timestamp)
		assert.Contains(t, event.Error, "simulated API server write failure")
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for agent.provisioning.failed event")
	}
}
