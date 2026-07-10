package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// AgentRole represents the topology role an agent fulfills within a community.
// Role is a behavioral property of a community assignment, not an intrinsic
// property of the Agent aggregate.
type AgentRole string

const (
	// AgentRoleStandalone is the sole role permitted in a single-agent topology community.
	AgentRoleStandalone AgentRole = "standalone"
	// AgentRoleHub is the coordinator role in a hub-spoke topology community.
	AgentRoleHub AgentRole = "hub"
	// AgentRoleSpoke is the worker role in a hub-spoke topology community.
	AgentRoleSpoke AgentRole = "spoke"
)

// CommunityAssignment is the aggregate that records an agent's membership in a
// community together with its topology role. It replaces the deprecated Role and
// CommunityID fields that previously lived on the Agent aggregate.
type CommunityAssignment struct {
	CommunityID uuid.UUID  `json:"community_id"`
	AgentID     uuid.UUID  `json:"agent_id"`
	TenantID    string     `json:"tenant_id"`
	Role        AgentRole  `json:"role"`
	InformedAt  *time.Time `json:"informed_at,omitempty"` // nil = stale / not yet reconciled
	AssignedAt  time.Time  `json:"assigned_at"`
}

// Validate checks the business invariants of the CommunityAssignment aggregate.
func (ca CommunityAssignment) Validate() error {
	if ca.CommunityID == uuid.Nil {
		return errors.New("community_id is required")
	}
	if ca.AgentID == uuid.Nil {
		return errors.New("agent_id is required")
	}
	if ca.TenantID == "" {
		return errors.New("tenant_id is required")
	}
	if ca.Role != AgentRoleStandalone && ca.Role != AgentRoleHub && ca.Role != AgentRoleSpoke {
		return errors.New("invalid agent role")
	}
	if ca.AssignedAt.IsZero() {
		return errors.New("assigned_at is required")
	}
	return nil
}
