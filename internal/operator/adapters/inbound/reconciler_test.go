package inbound_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/morphy76/tacito-square/internal/operator/adapters/inbound"
	"github.com/morphy76/tacito-square/pkg/kubernetes/apis/tacito/v1alpha1"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// mockReconcileService mocks the ReconcileAgentService interface.
type mockReconcileService struct {
	reconcileFunc func(ctx context.Context, agent *v1alpha1.TacitoAgent) error
}

func (m *mockReconcileService) Reconcile(ctx context.Context, agent *v1alpha1.TacitoAgent) error {
	if m.reconcileFunc != nil {
		return m.reconcileFunc(ctx, agent)
	}
	return nil
}

func (m *mockReconcileService) BuildDeployment(ctx context.Context, agent *v1alpha1.TacitoAgent) (*appsv1.Deployment, error) {
	return nil, nil
}

func (m *mockReconcileService) BuildHeadlessService(ctx context.Context, agent *v1alpha1.TacitoAgent) (*corev1.Service, error) {
	return nil, nil
}

func TestNewTacitoAgentReconciler(t *testing.T) {
	scheme := runtime.NewScheme()
	err := v1alpha1.AddToScheme(scheme)
	require.NoError(t, err)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	mockSvc := &mockReconcileService{}
	logBuf := &bytes.Buffer{}
	logger := zerolog.New(logBuf)

	r := inbound.NewTacitoAgentReconciler(fakeClient, scheme, mockSvc, logger)
	assert.NotNil(t, r)
}

func TestReconcile_NotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	err := v1alpha1.AddToScheme(scheme)
	require.NoError(t, err)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	mockSvc := &mockReconcileService{
		reconcileFunc: func(ctx context.Context, agent *v1alpha1.TacitoAgent) error {
			t.Fatal("service should not be called when resource is not found")
			return nil
		},
	}
	logBuf := &bytes.Buffer{}
	logger := zerolog.New(logBuf).Level(zerolog.InfoLevel)

	r := inbound.NewTacitoAgentReconciler(fakeClient, scheme, mockSvc, logger)

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: "default",
			Name:      "non-existent-agent",
		},
	}

	result, err := r.Reconcile(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)

	// Assert that a structured JSON log was emitted for not-found resource
	assert.Contains(t, logBuf.String(), "TacitoAgent resource not found, ignoring since object must be deleted")
	assert.Contains(t, logBuf.String(), "\"namespace\":\"default\"")
	assert.Contains(t, logBuf.String(), "\"name\":\"non-existent-agent\"")
}

func TestReconcile_Success(t *testing.T) {
	scheme := runtime.NewScheme()
	err := v1alpha1.AddToScheme(scheme)
	require.NoError(t, err)

	agent := &v1alpha1.TacitoAgent{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "my-agent",
		},
		Spec: v1alpha1.TacitoAgentSpec{
			TenantID:     "tenant-123",
			AgentName:    "agent-foo",
			CommunityRef: "community-abc",
			LLMConfig: v1alpha1.LLMConfig{
				Model: "gpt-4o",
			},
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(agent).Build()

	called := false
	mockSvc := &mockReconcileService{
		reconcileFunc: func(ctx context.Context, gotAgent *v1alpha1.TacitoAgent) error {
			called = true
			assert.Equal(t, agent.Name, gotAgent.Name)
			assert.Equal(t, agent.Namespace, gotAgent.Namespace)
			assert.Equal(t, "tenant-123", gotAgent.Spec.TenantID)
			assert.Equal(t, "agent-foo", gotAgent.Spec.AgentName)
			return nil
		},
	}

	logger := zerolog.Nop()
	r := inbound.NewTacitoAgentReconciler(fakeClient, scheme, mockSvc, logger)

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: "default",
			Name:      "my-agent",
		},
	}

	result, err := r.Reconcile(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
	assert.True(t, called)
}

func TestReconcile_ServiceError(t *testing.T) {
	scheme := runtime.NewScheme()
	err := v1alpha1.AddToScheme(scheme)
	require.NoError(t, err)

	agent := &v1alpha1.TacitoAgent{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "my-agent",
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(agent).Build()

	expectedErr := errors.New("something went wrong in the service layer")
	mockSvc := &mockReconcileService{
		reconcileFunc: func(ctx context.Context, gotAgent *v1alpha1.TacitoAgent) error {
			return expectedErr
		},
	}

	logBuf := &bytes.Buffer{}
	logger := zerolog.New(logBuf).Level(zerolog.ErrorLevel)
	r := inbound.NewTacitoAgentReconciler(fakeClient, scheme, mockSvc, logger)

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: "default",
			Name:      "my-agent",
		},
	}

	result, err := r.Reconcile(context.Background(), req)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Equal(t, ctrl.Result{}, result)

	// Assert error log contains structured details
	assert.Contains(t, logBuf.String(), "failed to reconcile TacitoAgent through application service")
	assert.Contains(t, logBuf.String(), "something went wrong in the service layer")
}
