package events

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// DomainEvent is the canonical, schema-agnostic event envelope.
// Every event published or consumed in the system MUST conform to this structure.
type DomainEvent struct {
	// EventID is a globally unique identifier for this event instance (UUID v4).
	EventID string `json:"event_id"`

	// SchemaRef is the versioned URN identifying the event's payload schema.
	SchemaRef string `json:"schema_ref"`

	// Source identifies the component that emitted this event.
	Source string `json:"source"`

	// TenantID is the resolved tenant identifier for this event.
	TenantID string `json:"tenant_id"`

	// OccurredAt is the UTC timestamp when the event was created (RFC3339Nano).
	OccurredAt string `json:"occurred_at"`

	// Payload contains the schema-specific event data.
	Payload json.RawMessage `json:"payload"`
}

// NewDomainEvent constructs a new DomainEvent with auto-populated system fields.
// Returns DomainEvent by value.
func NewDomainEvent(schemaRef, source, tenantID string, payload any) (DomainEvent, error) {
	if schemaRef == "" {
		return DomainEvent{}, errors.New("schemaRef must not be empty")
	}
	if source == "" {
		return DomainEvent{}, errors.New("source must not be empty")
	}
	if tenantID == "" {
		return DomainEvent{}, errors.New("tenantID must not be empty")
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return DomainEvent{}, err
	}

	return DomainEvent{
		EventID:    uuid.New().String(),
		SchemaRef:  schemaRef,
		Source:     source,
		TenantID:   tenantID,
		OccurredAt: time.Now().UTC().Format(time.RFC3339Nano),
		Payload:    payloadBytes,
	}, nil
}
