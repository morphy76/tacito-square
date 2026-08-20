package crd_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"



	"github.com/google/uuid"
	crdadapter "github.com/morphy76/tacito-square/internal/keeper/adapters/outbound/crd"
	"github.com/morphy76/tacito-square/internal/keeper/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
	"github.com/morphy76/tacito-square/pkg/kubernetes/apis/tacito/v1alpha1"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
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
	err = corev1.AddToScheme(scheme)
	require.NoError(t, err)

	bindingID := uuid.New()
	llmRepo := newMockLLMBindingRepository()
	llmRepo.bindings[bindingID] = &model.LLMBinding{
		ID:              bindingID,
		TenantID:        "tenant-1",
		APIBaseURL:      "https://custom-endpoint.com/v1",
		APIKeySecretRef: "my-zitadel-secret",
		DefaultModel:    "gpt-4",
		Status:          model.StatusActive,
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	coordinator := crdadapter.NewK8sCRDCoordinatorWithClient(fakeClient, "tacito", llmRepo, nil, nil, nil, nil)

	agentID := uuid.New()
	communityID := uuid.New()
	agent := &model.Agent{
		ID:          agentID,
		TenantID:    "tenant-1",
		Name:        "agent-1",
		Description: "A helpful assistant",
		Brain: model.BrainConfig{
			LLMBindingID: bindingID,
			Temperature:  ptrFloat64(0.5),
			MaxTokens:    ptrInt(1000),
		},
		Prompts:           []uuid.UUID{},
		PromptCollections: []uuid.UUID{},
		CommunityID:       &communityID,
	}

	err = coordinator.SubmitAgentCRD(context.Background(), agent)
	assert.NoError(t, err)

	// Fetch custom resource from fake client
	fetched := &v1alpha1.TacitoAgent{}
	key := types.NamespacedName{Namespace: "tacito", Name: "u-" + agentID.String()}
	err = fakeClient.Get(context.Background(), key, fetched)
	assert.NoError(t, err)

	// Assert mapped values
	assert.Equal(t, "tenant-1", fetched.Spec.TenantID)
	assert.Equal(t, "agent-1", fetched.Spec.AgentName)
	assert.Equal(t, communityID.String(), fetched.Spec.CommunityRef)
	assert.Equal(t, "gpt-4", fetched.Spec.LLMConfig.Model)
	assert.Equal(t, "0.5", *fetched.Spec.LLMConfig.Temperature)
	assert.Equal(t, int32(1000), *fetched.Spec.LLMConfig.MaxTokens)
	assert.Equal(t, "https://custom-endpoint.com/v1", *fetched.Spec.LLMConfig.Endpoint)
	assert.Equal(t, "my-zitadel-secret", *fetched.Spec.LLMConfig.CredentialsSecret)
}

func TestSubmitAgentCRD_UpdateSuccess(t *testing.T) {
	scheme := runtime.NewScheme()
	err := v1alpha1.AddToScheme(scheme)
	require.NoError(t, err)
	err = corev1.AddToScheme(scheme)
	require.NoError(t, err)

	agentID := uuid.New()
	communityID := uuid.New()

	// Pre-populate an existing CRD
	existing := &v1alpha1.TacitoAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "u-" + agentID.String(),
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

	bindingID := uuid.New()
	llmRepo := newMockLLMBindingRepository()
	llmRepo.bindings[bindingID] = &model.LLMBinding{
		ID:              bindingID,
		TenantID:        "tenant-1",
		APIBaseURL:      "https://new-endpoint.com/v1",
		APIKeySecretRef: "new-zitadel-secret",
		DefaultModel:    "gpt-4",
		Status:          model.StatusActive,
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existing).Build()
	coordinator := crdadapter.NewK8sCRDCoordinatorWithClient(fakeClient, "tacito", llmRepo, nil, nil, nil, nil)

	// Mapped model updates name and model
	agent := &model.Agent{
		ID:          agentID,
		TenantID:    "tenant-1",
		Name:        "new-agent-name",
		Description: "A helpful assistant",
		Brain: model.BrainConfig{
			LLMBindingID: bindingID,
			Temperature:  ptrFloat64(0.7),
			MaxTokens:    ptrInt(2000),
		},
		Prompts:           []uuid.UUID{},
		PromptCollections: []uuid.UUID{},
		CommunityID:       &communityID,
	}

	err = coordinator.SubmitAgentCRD(context.Background(), agent)
	assert.NoError(t, err)

	// Fetch custom resource from fake client and check updates
	fetched := &v1alpha1.TacitoAgent{}
	key := types.NamespacedName{Namespace: "tacito", Name: "u-" + agentID.String()}
	err = fakeClient.Get(context.Background(), key, fetched)
	assert.NoError(t, err)

	assert.Equal(t, "new-agent-name", fetched.Spec.AgentName)
	assert.Equal(t, "gpt-4", fetched.Spec.LLMConfig.Model)
	assert.Equal(t, "0.7", *fetched.Spec.LLMConfig.Temperature)
	assert.Equal(t, int32(2000), *fetched.Spec.LLMConfig.MaxTokens)
	assert.Equal(t, "https://new-endpoint.com/v1", *fetched.Spec.LLMConfig.Endpoint)
	assert.Equal(t, "new-zitadel-secret", *fetched.Spec.LLMConfig.CredentialsSecret)
}

func TestSubmitAgentCRD_ConflictResolution(t *testing.T) {
	scheme := runtime.NewScheme()
	err := v1alpha1.AddToScheme(scheme)
	require.NoError(t, err)
	err = corev1.AddToScheme(scheme)
	require.NoError(t, err)

	agentID := uuid.New()
	communityID := uuid.New()

	// Pre-populate an existing CRD
	existing := &v1alpha1.TacitoAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "u-" + agentID.String(),
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

	coordinator := crdadapter.NewK8sCRDCoordinatorWithClient(fakeClient, "tacito", newMockLLMBindingRepository(), nil, nil, nil, nil)

	agent := &model.Agent{
		ID:          agentID,
		TenantID:    "tenant-1",
		Name:        "agent-new",
		Description: "A helpful assistant",
		Brain: model.BrainConfig{
			LLMBindingID: uuid.New(),
			Temperature:  ptrFloat64(0.5),
			MaxTokens:    ptrInt(1000),
		},
		CommunityID: &communityID,
	}

	err = coordinator.SubmitAgentCRD(context.Background(), agent)
	assert.NoError(t, err)

	// Assert conflict update was retried and completed successfully
	assert.Equal(t, 2, updateAttempts)

	// Verify updates persisted
	fetched := &v1alpha1.TacitoAgent{}
	key := types.NamespacedName{Namespace: "tacito", Name: "u-" + agentID.String()}
	err = fakeClient.Get(context.Background(), key, fetched)
	assert.NoError(t, err)
	assert.Equal(t, "agent-new", fetched.Spec.AgentName)
}

func TestSubmitAgentCRD_Timeout(t *testing.T) {
	scheme := runtime.NewScheme()
	err := v1alpha1.AddToScheme(scheme)
	require.NoError(t, err)
	err = corev1.AddToScheme(scheme)
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

	coordinator := crdadapter.NewK8sCRDCoordinatorWithClient(fakeClient, "tacito", newMockLLMBindingRepository(), nil, nil, nil, nil)

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
	err = corev1.AddToScheme(scheme)
	require.NoError(t, err)

	agentID := uuid.New()
	existing := &v1alpha1.TacitoAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "u-" + agentID.String(),
			Namespace: "tacito",
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existing).Build()
	coordinator := crdadapter.NewK8sCRDCoordinatorWithClient(fakeClient, "tacito", newMockLLMBindingRepository(), nil, nil, nil, nil)

	agent := &model.Agent{
		ID: agentID,
	}

	// Delete existing
	err = coordinator.TeardownAgentCRD(context.Background(), agent)
	assert.NoError(t, err)

	// Assert deleted
	fetched := &v1alpha1.TacitoAgent{}
	key := types.NamespacedName{Namespace: "tacito", Name: "u-" + agentID.String()}
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

type mockPromptRepository struct {
	outbound.PromptRepository
	templates   map[uuid.UUID]*model.PromptTemplate
	collections map[uuid.UUID][]*model.PromptTemplate
	getErr      error
}

func (m *mockPromptRepository) GetTemplateByID(ctx context.Context, id uuid.UUID) (*model.PromptTemplate, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	t, ok := m.templates[id]
	if !ok {
		return nil, fmt.Errorf("prompt template not found: %s", id)
	}
	return t, nil
}

func (m *mockPromptRepository) ResolveCollectionPrompts(ctx context.Context, collectionID uuid.UUID) ([]*model.PromptTemplate, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.collections[collectionID], nil
}

type mockSkillRepository struct {
	outbound.SkillRepository
	skills map[uuid.UUID]*model.Skill
	getErr error
}

func (m *mockSkillRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Skill, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	s, ok := m.skills[id]
	if !ok {
		return nil, fmt.Errorf("skill not found: %s", id)
	}
	return s, nil
}

type mockMCPClientRepository struct {
	outbound.MCPClientRepository
	clients map[uuid.UUID]*model.MCPClient
	getErr  error
}

func (m *mockMCPClientRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.MCPClient, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	c, ok := m.clients[id]
	if !ok {
		return nil, fmt.Errorf("mcp client not found: %s", id)
	}
	return c, nil
}

type mockLLMBindingRepository struct {
	outbound.LLMBindingRepository
	bindings map[uuid.UUID]*model.LLMBinding
	getErr   error
}

func newMockLLMBindingRepository() *mockLLMBindingRepository {
	return &mockLLMBindingRepository{
		bindings: make(map[uuid.UUID]*model.LLMBinding),
	}
}

func (m *mockLLMBindingRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.LLMBinding, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	b, ok := m.bindings[id]
	if !ok {
		return &model.LLMBinding{
			ID:              id,
			TenantID:        "tenant-1",
			APIBaseURL:      "https://custom-endpoint.com/v1",
			APIKeySecretRef: "my-zitadel-secret",
			DefaultModel:    "gpt-4",
			Status:          model.StatusActive,
		}, nil
	}
	return b, nil
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
	err = corev1.AddToScheme(scheme)
	require.NoError(t, err)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	coordinator := crdadapter.NewK8sCRDCoordinatorWithClient(fakeClient, "tacito", newMockLLMBindingRepository(), nil, nil, nil, nc)

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
	defer func() { _ = sub.Unsubscribe() }()

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
	err = corev1.AddToScheme(scheme)
	require.NoError(t, err)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	coordinator := crdadapter.NewK8sCRDCoordinatorWithClient(fakeClient, "tacito", newMockLLMBindingRepository(), nil, nil, nil, nc)

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
	defer func() { _ = sub.Unsubscribe() }()

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
	err = corev1.AddToScheme(scheme)
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

	coordinator := crdadapter.NewK8sCRDCoordinatorWithClient(fakeClient, "tacito", newMockLLMBindingRepository(), nil, nil, nil, nc)

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
	defer func() { _ = sub.Unsubscribe() }()

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

func TestResolveAndSynthesizeSystemPrompt_Success(t *testing.T) {
	promptID := uuid.New()
	skillID1 := uuid.New()
	skillID2 := uuid.New()

	promptRepo := &mockPromptRepository{
		templates: map[uuid.UUID]*model.PromptTemplate{
			promptID: {
				ID:      promptID,
				Content: "Always be professional.",
				Status:  model.PromptStatusActive,
			},
		},
		collections: make(map[uuid.UUID][]*model.PromptTemplate),
	}
	skillRepo := &mockSkillRepository{
		skills: map[uuid.UUID]*model.Skill{
			skillID1: {
				ID:          skillID1,
				Name:        "WebSearch",
				Description: "Search the web",
				Content:     "WebSearch guidelines",
			},
			skillID2: {
				ID:          skillID2,
				Name:        "Calculations",
				Description: "Perform math operations",
				Content:     "Calculations guidelines",
			},
		},
	}

	coordinator := crdadapter.NewK8sCRDCoordinatorWithClient(nil, "tacito", newMockLLMBindingRepository(), promptRepo, skillRepo, nil, nil)

	agent := &model.Agent{
		ID:                uuid.New(),
		Description:       "A helpful AI assistant",
		Prompts:           []uuid.UUID{promptID},
		PromptCollections: []uuid.UUID{},
		Skills:            []uuid.UUID{skillID1, skillID2},
	}

	synthesized, err := coordinator.ResolveAndSynthesizeSystemPrompt(context.Background(), agent, "")
	require.NoError(t, err)

	var config crdadapter.PropagatedAgentConfig
	err = json.Unmarshal([]byte(synthesized), &config)
	require.NoError(t, err)

	assert.Equal(t, "A helpful AI assistant", config.Description)
	assert.Equal(t, "Always be professional.", config.Directives)
	require.Len(t, config.Skills, 2)
	assert.Equal(t, "WebSearch", config.Skills[0].Name)
	assert.Equal(t, "Search the web", config.Skills[0].Description)
	assert.Equal(t, "WebSearch guidelines", config.Skills[0].Content)
	assert.Equal(t, "Calculations", config.Skills[1].Name)
	assert.Equal(t, "Perform math operations", config.Skills[1].Description)
}

func TestResolveAndSynthesizeSystemPrompt_HubAgentMerging_Success(t *testing.T) {
	businessPromptID := uuid.New()

	promptRepo := &mockPromptRepository{
		templates: map[uuid.UUID]*model.PromptTemplate{
			model.HubSystemPromptTemplateID: {
				ID:      model.HubSystemPromptTemplateID,
				Content: "Role instructions: {{.Description}} and spokes: {{.Spokes}}",
				Status:  model.PromptStatusActive,
			},
			businessPromptID: {
				ID:      businessPromptID,
				Content: "Business specific instructions.",
				Status:  model.PromptStatusActive,
			},
		},
		collections: make(map[uuid.UUID][]*model.PromptTemplate),
	}

	coordinator := crdadapter.NewK8sCRDCoordinatorWithClient(nil, "tacito", newMockLLMBindingRepository(), promptRepo, nil, nil, nil)

	agent := &model.Agent{
		ID:                uuid.New(),
		Description:       "A helpful assistant",
		Prompts:           []uuid.UUID{businessPromptID},
		PromptCollections: []uuid.UUID{},
	}

	synthesized, err := coordinator.ResolveAndSynthesizeSystemPrompt(context.Background(), agent, "hub")
	require.NoError(t, err)

	var config crdadapter.PropagatedAgentConfig
	err = json.Unmarshal([]byte(synthesized), &config)
	require.NoError(t, err)

	assert.Equal(t, "Role instructions: {{.Description}} and spokes: {{.Spokes}}\n\nBusiness specific instructions.", config.Directives)
	assert.Equal(t, "A helpful assistant", config.Description)
}

func TestSubmitAgentCRD_SynthesizedPromptAndTenantMapped(t *testing.T) {
	scheme := runtime.NewScheme()
	err := v1alpha1.AddToScheme(scheme)
	require.NoError(t, err)
	err = corev1.AddToScheme(scheme)
	require.NoError(t, err)

	promptID := uuid.New()
	skillID := uuid.New()
	promptRepo := &mockPromptRepository{
		templates: map[uuid.UUID]*model.PromptTemplate{
			promptID: {
				ID:      promptID,
				Content: "Be short.",
				Status:  model.PromptStatusActive,
			},
		},
		collections: make(map[uuid.UUID][]*model.PromptTemplate),
	}
	skillRepo := &mockSkillRepository{
		skills: map[uuid.UUID]*model.Skill{
			skillID: {
				ID:          skillID,
				Name:        "Math",
				Description: "Solves arithmetic",
				Content:     "Math guidelines",
			},
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	coordinator := crdadapter.NewK8sCRDCoordinatorWithClient(fakeClient, "tacito", newMockLLMBindingRepository(), promptRepo, skillRepo, nil, nil)
	agentID := uuid.New()
	agent := &model.Agent{
		ID:                agentID,
		TenantID:          "tenant-crd-tenant",
		Name:              "agent-crd",
		Description:       "Direct assistant",
		Prompts:           []uuid.UUID{promptID},
		PromptCollections: []uuid.UUID{},
		Skills:            []uuid.UUID{skillID},
		Brain: model.BrainConfig{
			LLMBindingID: uuid.New(),
		},
	}

	err = coordinator.SubmitAgentCRD(context.Background(), agent)
	assert.NoError(t, err)

	fetched := &v1alpha1.TacitoAgent{}
	key := types.NamespacedName{Namespace: "tacito", Name: "u-" + agentID.String()}
	err = fakeClient.Get(context.Background(), key, fetched)
	assert.NoError(t, err)

	assert.Equal(t, "tenant-crd-tenant", fetched.Spec.TenantID)

	var config crdadapter.PropagatedAgentConfig
	err = json.Unmarshal([]byte(fetched.Spec.SystemPrompt), &config)
	require.NoError(t, err)

	assert.Equal(t, "Direct assistant", config.Description)
	assert.Equal(t, "Be short.", config.Directives)
	require.Len(t, config.Skills, 1)
	assert.Equal(t, "Math", config.Skills[0].Name)
	assert.Equal(t, "Solves arithmetic", config.Skills[0].Description)
	assert.Equal(t, "Math guidelines", config.Skills[0].Content)
}

func TestResolveAndSynthesizeSystemPrompt_MissingResources(t *testing.T) {
	promptID := uuid.New()
	skillID := uuid.New()

	// Missing template
	promptRepoMissing := &mockPromptRepository{
		templates:   map[uuid.UUID]*model.PromptTemplate{},
		collections: make(map[uuid.UUID][]*model.PromptTemplate),
	}
	skillRepo := &mockSkillRepository{
		skills: map[uuid.UUID]*model.Skill{
			skillID: {
				ID:   skillID,
				Name: "Math",
			},
		},
	}

	coordinator := crdadapter.NewK8sCRDCoordinatorWithClient(nil, "tacito", newMockLLMBindingRepository(), promptRepoMissing, skillRepo, nil, nil)
	agent := &model.Agent{
		ID:                uuid.New(),
		Prompts:           []uuid.UUID{promptID},
		PromptCollections: []uuid.UUID{},
	}

	_, err := coordinator.ResolveAndSynthesizeSystemPrompt(context.Background(), agent, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "prompt template not found")

	// Missing skill
	promptRepo := &mockPromptRepository{
		templates: map[uuid.UUID]*model.PromptTemplate{
			promptID: {
				ID:     promptID,
				Status: model.PromptStatusActive,
			},
		},
		collections: make(map[uuid.UUID][]*model.PromptTemplate),
	}
	skillRepoMissing := &mockSkillRepository{
		skills: map[uuid.UUID]*model.Skill{},
	}

	coordinator = crdadapter.NewK8sCRDCoordinatorWithClient(nil, "tacito", newMockLLMBindingRepository(), promptRepo, skillRepoMissing, nil, nil)
	agent = &model.Agent{
		ID:                uuid.New(),
		Prompts:           []uuid.UUID{promptID},
		PromptCollections: []uuid.UUID{},
		Skills:            []uuid.UUID{skillID},
	}

	_, err = coordinator.ResolveAndSynthesizeSystemPrompt(context.Background(), agent, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "skill not found")
}

func TestGetAgentCRDStatus_Existing(t *testing.T) {
	scheme := runtime.NewScheme()
	err := v1alpha1.AddToScheme(scheme)
	require.NoError(t, err)
	err = corev1.AddToScheme(scheme)
	require.NoError(t, err)

	agentID := uuid.New()
	existing := &v1alpha1.TacitoAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "u-" + agentID.String(),
			Namespace: "tacito",
		},
		Status: v1alpha1.TacitoAgentStatus{
			Phase:    v1alpha1.PhaseRunning,
			Replicas: 2,
			Conditions: []metav1.Condition{
				{
					Type:    "Ready",
					Status:  metav1.ConditionTrue,
					Reason:  "PodRunning",
					Message: "Pod healthy",
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existing).Build()
	coordinator := crdadapter.NewK8sCRDCoordinatorWithClient(fakeClient, "tacito", newMockLLMBindingRepository(), nil, nil, nil, nil)

	status, err := coordinator.GetAgentCRDStatus(context.Background(), agentID)
	assert.NoError(t, err)
	require.NotNil(t, status)
	assert.Equal(t, v1alpha1.PhaseRunning, status.Phase)
	assert.Equal(t, int32(2), status.Replicas)
	require.Len(t, status.Conditions, 1)
	assert.Equal(t, "Ready", status.Conditions[0].Type)
}

func TestGetAgentCRDStatus_NonExistent(t *testing.T) {
	scheme := runtime.NewScheme()
	err := v1alpha1.AddToScheme(scheme)
	require.NoError(t, err)
	err = corev1.AddToScheme(scheme)
	require.NoError(t, err)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	coordinator := crdadapter.NewK8sCRDCoordinatorWithClient(fakeClient, "tacito", newMockLLMBindingRepository(), nil, nil, nil, nil)

	agentID := uuid.New()
	status, err := coordinator.GetAgentCRDStatus(context.Background(), agentID)
	assert.NoError(t, err)
	assert.Nil(t, status)
}

func TestGetAgentCRDStatus_Timeout(t *testing.T) {
	scheme := runtime.NewScheme()
	err := v1alpha1.AddToScheme(scheme)
	require.NoError(t, err)
	err = corev1.AddToScheme(scheme)
	require.NoError(t, err)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithInterceptorFuncs(interceptor.Funcs{
			Get: func(ctx context.Context, cl client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				select {
				case <-time.After(10 * time.Millisecond):
					return cl.Get(ctx, key, obj, opts...)
				case <-ctx.Done():
					return ctx.Err()
				}
			},
		}).
		Build()

	coordinator := crdadapter.NewK8sCRDCoordinatorWithClient(fakeClient, "tacito", newMockLLMBindingRepository(), nil, nil, nil, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = coordinator.GetAgentCRDStatus(ctx, uuid.New())
	assert.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded))
}

func TestSubmitAgentCRD_WithMCPClients(t *testing.T) {
	scheme := runtime.NewScheme()
	err := v1alpha1.AddToScheme(scheme)
	require.NoError(t, err)
	err = corev1.AddToScheme(scheme)
	require.NoError(t, err)

	clientID := uuid.New()
	mcpRepo := &mockMCPClientRepository{
		clients: map[uuid.UUID]*model.MCPClient{
			clientID: {
				ID:        clientID,
				TenantID:  "tenant-1",
				Name:      "my-mcp-client",
				Transport: model.TransportStdio,
				Command:   "node",
				Args:      []string{"server.js"},
				Env:       map[string]string{"ENV_VAR": "value"},
				Status:    model.MCPClientStatusActive,
			},
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	coordinator := crdadapter.NewK8sCRDCoordinatorWithClient(fakeClient, "tacito", newMockLLMBindingRepository(), nil, nil, mcpRepo, nil)

	agentID := uuid.New()
	agent := &model.Agent{
		ID:       agentID,
		TenantID: "tenant-1",
		Name:     "agent-with-mcp",
		Brain: model.BrainConfig{
			LLMBindingID: uuid.New(),
		},
		MCPClients: []model.MCPClientConfig{
			{
				ClientID:     clientID,
				AllowedTools: []string{"tool1", "tool2"},
				CustomEnv:    map[string]string{"ENV_VAR": "override", "NEW_VAR": "new"},
				CustomArgs:   []string{"extra-arg"},
			},
		},
	}

	err = coordinator.SubmitAgentCRD(context.Background(), agent)
	assert.NoError(t, err)

	fetched := &v1alpha1.TacitoAgent{}
	key := types.NamespacedName{Namespace: "tacito", Name: "u-" + agentID.String()}
	err = fakeClient.Get(context.Background(), key, fetched)
	assert.NoError(t, err)

	require.Len(t, fetched.Spec.MCPClients, 1)
	spec := fetched.Spec.MCPClients[0]
	assert.Equal(t, "my-mcp-client", spec.Name)
	assert.Equal(t, "stdio", spec.Transport)
	assert.Equal(t, "node", spec.Command)
	// Base args + custom args
	assert.Equal(t, []string{"server.js", "extra-arg"}, spec.Args)
	// Base env + custom overrides
	assert.Equal(t, "override", spec.Env["ENV_VAR"])
	assert.Equal(t, "new", spec.Env["NEW_VAR"])
	// Allowed tools whitelisting
	assert.Equal(t, []string{"tool1", "tool2"}, spec.AllowedTools)
}

func TestSubmitAgentCRD_TierPropagated(t *testing.T) {
	scheme := runtime.NewScheme()
	err := v1alpha1.AddToScheme(scheme)
	require.NoError(t, err)
	err = corev1.AddToScheme(scheme)
	require.NoError(t, err)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	coordinator := crdadapter.NewK8sCRDCoordinatorWithClient(fakeClient, "tacito", newMockLLMBindingRepository(), nil, nil, nil, nil)

	agentID := uuid.New()
	agent := &model.Agent{
		ID:       agentID,
		TenantID: "tenant-1",
		Name:     "tier-agent",
		Tier:     "heavy",
		Brain: model.BrainConfig{
			LLMBindingID: uuid.New(),
		},
	}

	err = coordinator.SubmitAgentCRD(context.Background(), agent)
	assert.NoError(t, err)

	fetched := &v1alpha1.TacitoAgent{}
	key := types.NamespacedName{Namespace: "tacito", Name: "u-" + agentID.String()}
	err = fakeClient.Get(context.Background(), key, fetched)
	assert.NoError(t, err)

	assert.Equal(t, "heavy", fetched.Spec.Tier)
}

func TestSubmitAgentCRD_EmptyTierPropagated(t *testing.T) {
	scheme := runtime.NewScheme()
	err := v1alpha1.AddToScheme(scheme)
	require.NoError(t, err)
	err = corev1.AddToScheme(scheme)
	require.NoError(t, err)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	coordinator := crdadapter.NewK8sCRDCoordinatorWithClient(fakeClient, "tacito", newMockLLMBindingRepository(), nil, nil, nil, nil)

	agentID := uuid.New()
	agent := &model.Agent{
		ID:       agentID,
		TenantID: "tenant-1",
		Name:     "default-tier-agent",
		// Tier intentionally empty
		Brain: model.BrainConfig{
			LLMBindingID: uuid.New(),
		},
	}

	err = coordinator.SubmitAgentCRD(context.Background(), agent)
	assert.NoError(t, err)

	fetched := &v1alpha1.TacitoAgent{}
	key := types.NamespacedName{Namespace: "tacito", Name: "u-" + agentID.String()}
	err = fakeClient.Get(context.Background(), key, fetched)
	assert.NoError(t, err)

	assert.Equal(t, "", fetched.Spec.Tier)
}

func TestSubmitAgentCRD_HubRolePropagated(t *testing.T) {
	scheme := runtime.NewScheme()
	err := v1alpha1.AddToScheme(scheme)
	require.NoError(t, err)
	err = corev1.AddToScheme(scheme)
	require.NoError(t, err)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	coordinator := crdadapter.NewK8sCRDCoordinatorWithClient(fakeClient, "tacito", newMockLLMBindingRepository(), nil, nil, nil, nil)

	agentID := uuid.New()
	agent := &model.Agent{
		ID:       agentID,
		TenantID: "tenant-1",
		Name:     "hub-agent",
		Brain: model.BrainConfig{
			LLMBindingID: uuid.New(),
		},
	}

	err = coordinator.SubmitAgentCRD(context.Background(), agent)
	assert.NoError(t, err)

	fetched := &v1alpha1.TacitoAgent{}
	key := types.NamespacedName{Namespace: "tacito", Name: "u-" + agentID.String()}
	err = fakeClient.Get(context.Background(), key, fetched)
	assert.NoError(t, err)

	// Role is populated from CommunityAssignment at reconciliation time (SPEC-FR-M6.5.1)
	assert.Equal(t, "", fetched.Spec.Role)
}

func TestSubmitAgentCRD_SpokeRolePropagated(t *testing.T) {
	scheme := runtime.NewScheme()
	err := v1alpha1.AddToScheme(scheme)
	require.NoError(t, err)
	err = corev1.AddToScheme(scheme)
	require.NoError(t, err)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	coordinator := crdadapter.NewK8sCRDCoordinatorWithClient(fakeClient, "tacito", newMockLLMBindingRepository(), nil, nil, nil, nil)

	agentID := uuid.New()
	agent := &model.Agent{
		ID:       agentID,
		TenantID: "tenant-1",
		Name:     "spoke-agent",
		Brain: model.BrainConfig{
			LLMBindingID: uuid.New(),
		},
	}

	err = coordinator.SubmitAgentCRD(context.Background(), agent)
	assert.NoError(t, err)

	fetched := &v1alpha1.TacitoAgent{}
	key := types.NamespacedName{Namespace: "tacito", Name: "u-" + agentID.String()}
	err = fakeClient.Get(context.Background(), key, fetched)
	assert.NoError(t, err)

	// Role is populated from CommunityAssignment at reconciliation time (SPEC-FR-M6.5.1)
	assert.Equal(t, "", fetched.Spec.Role)
}

func TestSubmitAgentCRD_RoleUpdated(t *testing.T) {
	scheme := runtime.NewScheme()
	err := v1alpha1.AddToScheme(scheme)
	require.NoError(t, err)
	err = corev1.AddToScheme(scheme)
	require.NoError(t, err)

	agentID := uuid.New()

	// Pre-populate an existing CRD with role "spoke"
	existing := &v1alpha1.TacitoAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "u-" + agentID.String(),
			Namespace: "tacito",
		},
		Spec: v1alpha1.TacitoAgentSpec{
			TenantID:  "tenant-1",
			AgentName: "updating-agent",
			Role:      "spoke",
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existing).Build()
	coordinator := crdadapter.NewK8sCRDCoordinatorWithClient(fakeClient, "tacito", newMockLLMBindingRepository(), nil, nil, nil, nil)

	// Update model — role now comes from CommunityAssignment, not agent.Role
	agent := &model.Agent{
		ID:       agentID,
		TenantID: "tenant-1",
		Name:     "updating-agent",
		Brain: model.BrainConfig{
			LLMBindingID: uuid.New(),
		},
	}

	err = coordinator.SubmitAgentCRD(context.Background(), agent)
	assert.NoError(t, err)

	// Fetch and verify update
	fetched := &v1alpha1.TacitoAgent{}
	key := types.NamespacedName{Namespace: "tacito", Name: "u-" + agentID.String()}
	err = fakeClient.Get(context.Background(), key, fetched)
	assert.NoError(t, err)

	// Role is populated from CommunityAssignment at reconciliation time (SPEC-FR-M6.5.1)
	assert.Equal(t, "", fetched.Spec.Role)
}

func ptrFloat64(v float64) *float64 { return &v }
func ptrInt(v int) *int { return &v }



