package events_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/pkg/events"
	"github.com/stretchr/testify/assert"
)

func TestNewDomainEvent_Success(t *testing.T) {
	schemaRef := "urn:tacito:schema:conversational:start-thread:v1"
	source := "keeper/pod-123"
	tenantID := "tenant-abc"
	payload := map[string]string{"foo": "bar"}

	evt, err := events.NewDomainEvent(schemaRef, source, tenantID, payload)
	assert.NoError(t, err)

	// Assert basic population
	assert.NotEmpty(t, evt.EventID)
	_, err = uuid.Parse(evt.EventID)
	assert.NoError(t, err, "EventID must be a valid UUID v4")

	assert.Equal(t, schemaRef, evt.SchemaRef)
	assert.Equal(t, source, evt.Source)
	assert.Equal(t, tenantID, evt.TenantID)

	// Assert OccurredAt
	assert.NotEmpty(t, evt.OccurredAt)
	parsedTime, err := time.Parse(time.RFC3339Nano, evt.OccurredAt)
	assert.NoError(t, err, "OccurredAt must be in RFC3339Nano format")
	assert.Equal(t, time.UTC, parsedTime.Location(), "OccurredAt must be in UTC")

	// Assert Payload serialization
	var unmarshalledPayload map[string]string
	err = json.Unmarshal(evt.Payload, &unmarshalledPayload)
	assert.NoError(t, err)
	assert.Equal(t, "bar", unmarshalledPayload["foo"])
}

func TestNewDomainEvent_ValidationErrors(t *testing.T) {
	schemaRef := "urn:tacito:schema:conversational:start-thread:v1"
	source := "keeper/pod-123"
	tenantID := "tenant-abc"
	payload := "test-payload"

	// Missing SchemaRef
	_, err := events.NewDomainEvent("", source, tenantID, payload)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "schemaRef")

	// Missing Source
	_, err = events.NewDomainEvent(schemaRef, "", tenantID, payload)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "source")

	// Missing TenantID
	_, err = events.NewDomainEvent(schemaRef, source, "", payload)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tenantID")
}

func TestNewDomainEvent_JSONMarshalError(t *testing.T) {
	schemaRef := "urn:tacito:schema:conversational:start-thread:v1"
	source := "keeper/pod-123"
	tenantID := "tenant-abc"

	// A channel cannot be marshalled to JSON
	badPayload := make(chan int)

	_, err := events.NewDomainEvent(schemaRef, source, tenantID, badPayload)
	assert.Error(t, err)
}
