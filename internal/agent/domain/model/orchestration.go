package model

import "errors"

// OrchestrationStatus represents the state of a thread orchestration.
type OrchestrationStatus string

const (
	StatusIdle         OrchestrationStatus = "idle"
	StatusWaitingSpoke OrchestrationStatus = "waiting_spoke"
)

// OrchestrationState represents the active routing/orchestration state of a conversation thread.
type OrchestrationState struct {
	ThreadID        string              `json:"thread_id"`
	CommunityID     string              `json:"community_id"`
	Status          OrchestrationStatus `json:"status"`
	PendingSpokes   map[string]string   `json:"pending_spokes,omitempty"`
	OriginalEventID string              `json:"original_event_id,omitempty"`
	LoopCount       int                 `json:"loop_count"`
	MaxLoops        int                 `json:"max_loops"`
}

// Validate ensures standard orchestration state constraints are met.
func (s *OrchestrationState) Validate() error {
	if s.ThreadID == "" {
		return errors.New("thread_id must not be empty")
	}
	if s.CommunityID == "" {
		return errors.New("community_id must not be empty")
	}
	if s.Status == "" {
		return errors.New("status must not be empty")
	}
	return nil
}
