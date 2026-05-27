package crd

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/morphy76/tacito-square/internal/keeper/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
	"github.com/morphy76/tacito-square/pkg/kubernetes/apis/tacito/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// K8sCRDCoordinator implements outbound.CRDCoordinator driving ports.
type K8sCRDCoordinator struct {
	client    client.Client
	namespace string
}

var _ outbound.CRDCoordinator = (*K8sCRDCoordinator)(nil)

// NewK8sCRDCoordinator creates a new K8sCRDCoordinator with a real controller-runtime Client.
func NewK8sCRDCoordinator(config *rest.Config) (*K8sCRDCoordinator, error) {
	scheme := runtime.NewScheme()
	if err := v1alpha1.AddToScheme(scheme); err != nil {
		return nil, err
	}

	c, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}

	return &K8sCRDCoordinator{
		client:    c,
		namespace: "tacito",
	}, nil
}

// NewK8sCRDCoordinatorWithClient constructs the coordinator directly using a pre-configured client (convenient for testing).
func NewK8sCRDCoordinatorWithClient(c client.Client, namespace string) *K8sCRDCoordinator {
	if namespace == "" {
		namespace = "tacito"
	}
	return &K8sCRDCoordinator{
		client:    c,
		namespace: namespace,
	}
}

// SubmitAgentCRD constructs and registers a TacitoAgent custom resource in the K8s cluster.
func (c *K8sCRDCoordinator) SubmitAgentCRD(ctx context.Context, agent *model.Agent) error {
	deadlineCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// 1. Resolve LLMConfig values
	var temp *string
	if agent.Brain.Temperature != 0.0 {
		tStr := strconv.FormatFloat(agent.Brain.Temperature, 'f', -1, 64)
		temp = &tStr
	}

	var maxTokens *int32
	if agent.Brain.MaxTokens > 0 {
		mt := int32(agent.Brain.MaxTokens)
		maxTokens = &mt
	}

	var communityRef string
	if agent.CommunityID != nil {
		communityRef = agent.CommunityID.String()
	}

	key := types.NamespacedName{Namespace: c.namespace, Name: agent.ID.String()}
	existing := &v1alpha1.TacitoAgent{}

	err := c.client.Get(deadlineCtx, key, existing)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Construct a brand new Custom Resource
			crdObj := &v1alpha1.TacitoAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      agent.ID.String(),
					Namespace: c.namespace,
				},
				Spec: v1alpha1.TacitoAgentSpec{
					TenantID:     agent.TenantID,
					AgentName:    agent.Name,
					CommunityRef: communityRef,
					LLMConfig: v1alpha1.LLMConfig{
						Model:       agent.Brain.Model,
						Temperature: temp,
						MaxTokens:   maxTokens,
					},
					// SystemPrompt will be resolved out-of-band in TASK-M4.6.2
				},
			}
			return c.client.Create(deadlineCtx, crdObj)
		}
		return fmt.Errorf("getting TacitoAgent CRD: %w", err)
	}

	// 2. Resource exists: fetch and update within a conflict resolution loop
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		latest := &v1alpha1.TacitoAgent{}
		if err := c.client.Get(deadlineCtx, key, latest); err != nil {
			return err
		}

		latest.Spec.TenantID = agent.TenantID
		latest.Spec.AgentName = agent.Name
		latest.Spec.CommunityRef = communityRef
		latest.Spec.LLMConfig.Model = agent.Brain.Model
		latest.Spec.LLMConfig.Temperature = temp
		latest.Spec.LLMConfig.MaxTokens = maxTokens

		return c.client.Update(deadlineCtx, latest)
	})
}

// TeardownAgentCRD deletes the corresponding TacitoAgent custom resource safely.
func (c *K8sCRDCoordinator) TeardownAgentCRD(ctx context.Context, agent *model.Agent) error {
	deadlineCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	crdObj := &v1alpha1.TacitoAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      agent.ID.String(),
			Namespace: c.namespace,
		},
	}

	err := c.client.Delete(deadlineCtx, crdObj)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("deleting TacitoAgent CRD: %w", err)
	}

	return nil
}
