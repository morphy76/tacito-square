package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type CommunityStatus string

const (
	CommunityStatusCreated    CommunityStatus = "created"
	CommunityStatusActive     CommunityStatus = "active"
	CommunityStatusSuspended  CommunityStatus = "suspended"
	CommunityStatusTerminated CommunityStatus = "terminated"
)

type CommunityTopology string

const (
	CommunityTopologyHubSpoke CommunityTopology = "hub-spoke"
)

type Community struct {
	ID            uuid.UUID              `json:"id"`
	TenantID      string                 `json:"tenant_id"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	Topology      CommunityTopology      `json:"topology"`
	Configuration map[string]interface{} `json:"configuration"`
	Status        CommunityStatus        `json:"status"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

func (c Community) Validate() error {
	if c.ID == uuid.Nil {
		return errors.New("id is required")
	}
	if c.TenantID == "" {
		return errors.New("tenant id is required")
	}
	if c.Name == "" {
		return errors.New("name is required")
	}
	if c.Topology != CommunityTopologyHubSpoke {
		return errors.New("invalid or unsupported topology")
	}
	if c.Status != CommunityStatusCreated &&
		c.Status != CommunityStatusActive &&
		c.Status != CommunityStatusSuspended &&
		c.Status != CommunityStatusTerminated {
		return errors.New("invalid community status")
	}
	return nil
}
