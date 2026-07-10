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
	corev1 "k8s.io/api/core/v1"
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
	client         client.Client
	namespace      string
	llmBindingRepo outbound.LLMBindingRepository
	promptRepo     outbound.PromptRepository
	skillRepo      outbound.SkillRepository
	mcpRepo        outbound.MCPClientRepository
	natsConn       *nats.Conn
}

var _ outbound.CRDCoordinator = (*K8sCRDCoordinator)(nil)

// NewK8sCRDCoordinator creates a new K8sCRDCoordinator with a real controller-runtime Client.
func NewK8sCRDCoordinator(
	config *rest.Config,
	llmBindingRepo outbound.LLMBindingRepository,
	promptRepo outbound.PromptRepository,
	skillRepo outbound.SkillRepository,
	mcpRepo outbound.MCPClientRepository,
	nc *nats.Conn,
) (*K8sCRDCoordinator, error) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := v1alpha1.AddToScheme(scheme); err != nil {
		return nil, err
	}

	c, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}

	return &K8sCRDCoordinator{
		client:         c,
		namespace:      "tacito",
		llmBindingRepo: llmBindingRepo,
		promptRepo:     promptRepo,
		skillRepo:      skillRepo,
		mcpRepo:        mcpRepo,
		natsConn:       nc,
	}, nil
}

// NewK8sCRDCoordinatorWithClient constructs the coordinator directly using a pre-configured client (convenient for testing).
func NewK8sCRDCoordinatorWithClient(
	c client.Client,
	namespace string,
	llmBindingRepo outbound.LLMBindingRepository,
	promptRepo outbound.PromptRepository,
	skillRepo outbound.SkillRepository,
	mcpRepo outbound.MCPClientRepository,
	nc *nats.Conn,
) *K8sCRDCoordinator {
	if namespace == "" {
		namespace = "tacito"
	}
	return &K8sCRDCoordinator{
		client:         c,
		namespace:      namespace,
		llmBindingRepo: llmBindingRepo,
		promptRepo:     promptRepo,
		skillRepo:      skillRepo,
		mcpRepo:        mcpRepo,
		natsConn:       nc,
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
// The role parameter carries the agent's community-assignment role (hub, spoke, standalone) and is used
// to select the appropriate role-specific prompt template. It is no longer read from agent.Role.
func (c *K8sCRDCoordinator) ResolveAndSynthesizeSystemPrompt(ctx context.Context, agent *model.Agent, role string) (string, error) {
	var directives string
	description := agent.Description

	if role == "hub" {
		if c.promptRepo != nil {
			// Fetch the role-specific template for hub
			roleTpl, err := c.promptRepo.GetTemplateByID(ctx, model.HubSystemPromptTemplateID)
			if err != nil {
				return "", fmt.Errorf("fetching role-specific hub prompt template: %w", err)
			}
			directives = roleTpl.Content
		}

		// If a business-specific prompt template is provided (and is not the hub template itself)
		if agent.PromptTemplate != uuid.Nil && agent.PromptTemplate != model.HubSystemPromptTemplateID {
			if c.promptRepo != nil {
				businessTpl, err := c.promptRepo.GetTemplateByID(ctx, agent.PromptTemplate)
				if err != nil {
					return "", fmt.Errorf("fetching business-specific prompt template: %w", err)
				}
				if description != "" {
					description = description + "\n\n" + businessTpl.Content
				} else {
					description = businessTpl.Content
				}
			}
		}
	} else {
		// Non-hub agent (spoke or general)
		if agent.PromptTemplate != uuid.Nil && c.promptRepo != nil {
			tpl, err := c.promptRepo.GetTemplateByID(ctx, agent.PromptTemplate)
			if err != nil {
				return "", fmt.Errorf("fetching prompt template: %w", err)
			}
			directives = tpl.Content
		}
	}

	var skillsList []SkillConfig
	if c.skillRepo != nil {
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
	}

	config := PropagatedAgentConfig{
		Description: description,
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
	systemPrompt, err := c.ResolveAndSynthesizeSystemPrompt(ctx, agent, "")
	if err != nil {
		return fmt.Errorf("resolving and synthesizing system prompt: %w", err)
	}

	deadlineCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// 1. Resolve LLMConfig values by fetching LLMBinding
	if c.llmBindingRepo == nil {
		return fmt.Errorf("llmBindingRepo is nil in CRD coordinator")
	}
	llmBinding, err := c.llmBindingRepo.GetByID(ctx, agent.Brain.LLMBindingID)
	if err != nil {
		return fmt.Errorf("resolving llm binding %s: %w", agent.Brain.LLMBindingID, err)
	}

	modelName := llmBinding.DefaultModel

	var temp *string
	if agent.Brain.Temperature != nil {
		tStr := strconv.FormatFloat(*agent.Brain.Temperature, 'f', -1, 64)
		temp = &tStr
	} else {
		tStr := strconv.FormatFloat(llmBinding.DefaultTemperature, 'f', -1, 64)
		temp = &tStr
	}

	var maxTokensVal int32
	if agent.Brain.MaxTokens != nil {
		maxTokensVal = int32(*agent.Brain.MaxTokens)
	} else {
		maxTokensVal = int32(llmBinding.DefaultMaxTokens)
	}
	maxTokens := &maxTokensVal

	endpoint := &llmBinding.APIBaseURL

	var credsSecret *string
	if llmBinding.APIKeySecretRef != "" {
		credsSecret = &llmBinding.APIKeySecretRef
	}

	var communityRef string
	if agent.CommunityID != nil {
		communityRef = agent.CommunityID.String()
	}

	var mcpClientSpecs []v1alpha1.MCPClientSpec
	for _, mcpConfig := range agent.MCPClients {
		if c.mcpRepo != nil {
			clientCfg, err := c.mcpRepo.GetByID(ctx, mcpConfig.ClientID)
			if err != nil {
				return fmt.Errorf("resolving mcp client config %s: %w", mcpConfig.ClientID, err)
			}

			env := make(map[string]string)
			for k, v := range clientCfg.Env {
				env[k] = v
			}
			for k, v := range mcpConfig.CustomEnv {
				env[k] = v
			}

			args := append([]string(nil), clientCfg.Args...)
			args = append(args, mcpConfig.CustomArgs...)

			mcpClientSpecs = append(mcpClientSpecs, v1alpha1.MCPClientSpec{
				Name:         clientCfg.Name,
				Transport:    string(clientCfg.Transport),
				Command:      clientCfg.Command,
				Args:         args,
				Env:          env,
				URL:          clientCfg.URL,
				AllowedTools: mcpConfig.AllowedTools,
			})
		}
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
						Model:             modelName,
						Temperature:       temp,
						MaxTokens:         maxTokens,
						Endpoint:          endpoint,
						CredentialsSecret: credsSecret,
					},
					MCPClients: mcpClientSpecs,
					Tier:       agent.Tier,
					Role:       "", // populated from CommunityAssignment at reconciliation time (SPEC-FR-M6.5.1)
				},
			}
			err = c.client.Create(deadlineCtx, crdObj)
			if err != nil {
				return err
			}
			return c.reconcileCredentialsSecret(deadlineCtx, agent, crdObj)
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
		latest.Spec.LLMConfig.Model = modelName
		latest.Spec.LLMConfig.Temperature = temp
		latest.Spec.LLMConfig.MaxTokens = maxTokens
		latest.Spec.LLMConfig.Endpoint = endpoint
		latest.Spec.LLMConfig.CredentialsSecret = credsSecret
		latest.Spec.MCPClients = mcpClientSpecs
		latest.Spec.Tier = agent.Tier
		latest.Spec.Role = "" // populated from CommunityAssignment at reconciliation time (SPEC-FR-M6.5.1)

		err = c.client.Update(deadlineCtx, latest)
		if err != nil {
			return err
		}
		return c.reconcileCredentialsSecret(deadlineCtx, agent, latest)
	})
}

// reconcileCredentialsSecret creates or updates the Kubernetes Secret s-<agentId> containing the LLM API credentials.
func (c *K8sCRDCoordinator) reconcileCredentialsSecret(ctx context.Context, agent *model.Agent, owner *v1alpha1.TacitoAgent) error {
	return nil
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
