package events_test

import (
	"testing"

	"github.com/morphy76/tacito-square/pkg/events"
	"github.com/stretchr/testify/assert"
)

func TestInfrastructureConstantsExist(t *testing.T) {
	assert.Equal(t, "urn:tacito:schema:infrastructure:agent-heartbeat:v1", events.SchemaInfrastructureAgentHeartbeat)
}
