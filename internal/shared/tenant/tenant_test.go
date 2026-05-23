package tenant_test

import (
	"context"
	"testing"

	"github.com/morphy76/tacito-square/internal/shared/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Tenant.Validate / New -----------------------------------------------

func TestNew_ValidTenantIDOnly(t *testing.T) {
	ten, err := tenant.New("acme.com", "")
	require.NoError(t, err)
	assert.Equal(t, "acme.com", ten.TenantID)
	assert.Equal(t, "", ten.SubscriptionID)
}

func TestNew_ValidTenantIDWithSubscription(t *testing.T) {
	ten, err := tenant.New("acme.com", "pro")
	require.NoError(t, err)
	assert.Equal(t, "pro", ten.SubscriptionID)
}

func TestNew_ValidMultiLabelDomain(t *testing.T) {
	_, err := tenant.New("sub.acme.co.uk", "")
	assert.NoError(t, err)
}

func TestNew_ValidSingleLabel(t *testing.T) {
	_, err := tenant.New("localhost", "")
	assert.NoError(t, err)
}

func TestNew_EmptyTenantID_ReturnsError(t *testing.T) {
	_, err := tenant.New("", "")
	assert.ErrorContains(t, err, "tenantId is required")
}

func TestNew_InvalidTenantID_StartsWithHyphen(t *testing.T) {
	_, err := tenant.New("-acme.com", "")
	assert.ErrorContains(t, err, "tenantId must follow domain URL syntax")
}

func TestNew_InvalidTenantID_EndsWithDot(t *testing.T) {
	_, err := tenant.New("acme.com.", "")
	assert.ErrorContains(t, err, "tenantId must follow domain URL syntax")
}

func TestNew_InvalidTenantID_ContainsUnderscore(t *testing.T) {
	_, err := tenant.New("acme_corp.com", "")
	assert.ErrorContains(t, err, "tenantId must follow domain URL syntax")
}

func TestNew_InvalidSubscriptionID_StartsWithHyphen(t *testing.T) {
	_, err := tenant.New("acme.com", "-pro")
	assert.ErrorContains(t, err, "subscriptionId must be alphanumeric")
}

func TestNew_InvalidSubscriptionID_EndsWithHyphen(t *testing.T) {
	_, err := tenant.New("acme.com", "pro-")
	assert.ErrorContains(t, err, "subscriptionId must be alphanumeric")
}

// --- Tenant.FullName ------------------------------------------------------

func TestFullName_WithoutSubscription(t *testing.T) {
	ten, _ := tenant.New("acme.com", "")
	assert.Equal(t, "acme.com", ten.FullName())
}

func TestFullName_WithSubscription(t *testing.T) {
	ten, _ := tenant.New("acme.com", "pro")
	assert.Equal(t, "acme.com-pro", ten.FullName())
}

func TestFullName_MultiLabelDomainWithSubscription(t *testing.T) {
	ten, _ := tenant.New("my-org.io", "enterprise-2024")
	assert.Equal(t, "my-org.io-enterprise-2024", ten.FullName())
}

// --- Context propagation --------------------------------------------------

func TestContextRoundtrip(t *testing.T) {
	ten, _ := tenant.New("acme.com", "pro")
	ctx := tenant.ContextWithTenant(context.Background(), ten)

	got := tenant.FromContext(ctx)
	require.NotNil(t, got)
	assert.Equal(t, ten.FullName(), got.FullName())
}

func TestFromContext_Missing_ReturnsNil(t *testing.T) {
	got := tenant.FromContext(context.Background())
	assert.Nil(t, got)
}
