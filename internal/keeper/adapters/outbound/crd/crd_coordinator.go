package crd

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/keeper/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
	"github.com/morphy76/tacito-square/internal/shared/observability"
	"github.com/morphy76/tacito-square/pkg/kubernetes/apis/tacito/v1alpha1"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
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
	client     client.Client
	namespace  string
	promptRepo outbound.PromptRepository
	skillRepo  outbound.SkillRepository
	natsConn   *nats.Conn
}

var _ outbound.CRDCoordinator = (*K8sCRDCoordinator)(nil)

// NewK8sCRDCoordinator creates a new K8sCRDCoordinator with a real controller-runtime Client.
func NewK8sCRDCoordinator(
	config *rest.Config,
	promptRepo outbound.PromptRepository,
	skillRepo outbound.SkillRepository,
	nc *nats.Conn,
) (*K8sCRDCoordinator, error) {
	scheme := runtime.NewScheme()
	if err := v1alpha1.AddToScheme(scheme); err != nil {
		return nil, err
	}

	c, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}

	return &K8sCRDCoordinator{
		client:     c,
		namespace:  "tacito",
		promptRepo: promptRepo,
		skillRepo:  skillRepo,
		natsConn:   nc,
	}, nil
}

// NewK8sCRDCoordinatorWithClient constructs the coordinator directly using a pre-configured client (convenient for testing).
func NewK8sCRDCoordinatorWithClient(
	c client.Client,
	namespace string,
	promptRepo outbound.PromptRepository,
	skillRepo outbound.SkillRepository,
	nc *nats.Conn,
) *K8sCRDCoordinator {
	if namespace == "" {
		namespace = "tacito"
	}
	return &K8sCRDCoordinator{
		client:     c,
		namespace:  namespace,
		promptRepo: promptRepo,
		skillRepo:  skillRepo,
		natsConn:   nc,
	}
}

// ProvisioningEvent represents the structured JSON payload published to NATS.
type ProvisioningEvent struct {
	TenantID    string `json:"tenant_id"`
	AgentID     string `json:"agent_id"`
	CommunityID string `json:"community_id"`
	Timestamp   string `json:"timestamp"`
	Error       string `json:"error,omitempty"`
}

// PublishProvisioningEvent serializes and broadcasts provisioning transition events onto the NATS event bus.
func (c *K8sCRDCoordinator) PublishProvisioningEvent(ctx context.Context, subject string, agent *model.Agent, errVal error) {
	if c.natsConn == nil {
		return
	}

	var errMsg string
	if errVal != nil {
		errMsg = errVal.Error()
	}

	var communityID string
	if agent.CommunityID != nil {
		communityID = agent.CommunityID.String()
	}

	event := ProvisioningEvent{
		TenantID:    agent.TenantID,
		AgentID:     agent.ID.String(),
		CommunityID: communityID,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Error:       errMsg,
	}

	data, err := json.Marshal(event)
	if err != nil {
		logger := observability.WithContext(log.Logger, ctx)
		logger.Error().Err(err).Msg("failed to marshal provisioning event")
		return
	}

	if err := c.natsConn.Publish(subject, data); err != nil {
		logger := observability.WithContext(log.Logger, ctx)
		logger.Error().Err(err).Str("subject", subject).Msg("failed to publish NATS provisioning event")
		return
	}

	logger := observability.WithContext(log.Logger, ctx)
	logger.Debug().Str("subject", subject).Msg("successfully published NATS provisioning event")
}

// SkillConfig defines the propagated skill structure.
type SkillConfig struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Content     string `json:"content"`
}

// PropagatedAgentConfig defines the structured keeper-agent context format.
type PropagatedAgentConfig struct {
	Description string        `json:"description"`
	Directives  string        `json:"directives"`
	Skills      []SkillConfig `json:"skills"`
}

// ResolveAndSynthesizeSystemPrompt fetches templates and skills out-of-band and compiles them into a system prompt.
func (c *K8sCRDCoordinator) ResolveAndSynthesizeSystemPrompt(ctx context.Context, agent *model.Agent) (string, error) {
	var directives string
	if agent.PromptTemplate != uuid.Nil {
		tpl, err := c.promptRepo.GetTemplateByID(ctx, agent.PromptTemplate)
		if err != nil {
			return "", fmt.Errorf("fetching prompt template: %w", err)
		}
		directives = tpl.Content
	}

	var skillsList []SkillConfig
	for _, skillID := range agent.Skills {
		skill, err := c.skillRepo.GetByID(ctx, skillID)
		if err != nil {
			return "", fmt.Errorf("fetching skill: %w", err)
		}
		skillsList = append(skillsList, SkillConfig{
			Name:        skill.Name,
			Description: skill.Description,
			Content:     skill.Content,
		})
	}

	config := PropagatedAgentConfig{
		Description: agent.Description,
		Directives:  directives,
		Skills:      skillsList,
	}

	data, err := json.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("marshaling system prompt structured json: %w", err)
	}

	return string(data), nil
}

// SubmitAgentCRD constructs and registers a TacitoAgent custom resource in the K8s cluster.
func (c *K8sCRDCoordinator) SubmitAgentCRD(ctx context.Context, agent *model.Agent) (err error) {
	c.PublishProvisioningEvent(ctx, "agent.provisioning.started", agent, nil)
	defer func() {
		if err != nil {
			logger := observability.WithContext(log.Logger, ctx)
			logger.Error().Err(err).Msg("failed to submit agent crd")
			c.PublishProvisioningEvent(ctx, "agent.provisioning.failed", agent, err)
		} else {
			c.PublishProvisioningEvent(ctx, "agent.provisioning.completed", agent, nil)
		}
	}()

	// Synthesis out-of-band prompt template & skills list
	systemPrompt, err := c.ResolveAndSynthesizeSystemPrompt(ctx, agent)
	if err != nil {
		return fmt.Errorf("resolving and synthesizing system prompt: %w", err)
	}

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

	key := types.NamespacedName{Namespace: c.namespace, Name: "u-" + strings.ToLower(agent.ID.String())}
	existing := &v1alpha1.TacitoAgent{}

	getErr := c.client.Get(deadlineCtx, key, existing)
	if getErr != nil {
		if apierrors.IsNotFound(getErr) {
			// Construct a brand new Custom Resource
			crdObj := &v1alpha1.TacitoAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "u-" + strings.ToLower(agent.ID.String()),
					Namespace: c.namespace,
				},
				Spec: v1alpha1.TacitoAgentSpec{
					TenantID:     agent.TenantID,
					AgentName:    agent.Name,
					CommunityRef: communityRef,
					SystemPrompt: systemPrompt,
					LLMConfig: v1alpha1.LLMConfig{
						Model:       agent.Brain.Model,
						Temperature: temp,
						MaxTokens:   maxTokens,
					},
				},
			}
			return c.client.Create(deadlineCtx, crdObj)
		}
		return fmt.Errorf("getting TacitoAgent CRD: %w", getErr)
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
		latest.Spec.SystemPrompt = systemPrompt
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
			Name:      "u-" + strings.ToLower(agent.ID.String()),
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

// GetAgentCRDStatus retrieves the current observed status subresource of a TacitoAgent custom resource.
func (c *K8sCRDCoordinator) GetAgentCRDStatus(ctx context.Context, agentID uuid.UUID) (*v1alpha1.TacitoAgentStatus, error) {
	deadlineCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	key := types.NamespacedName{Namespace: c.namespace, Name: "u-" + strings.ToLower(agentID.String())}
	agentCRD := &v1alpha1.TacitoAgent{}

	err := c.client.Get(deadlineCtx, key, agentCRD)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting TacitoAgent CRD status: %w", err)
	}

	return &agentCRD.Status, nil
}
