// Package tenant defines the Tenant identity model shared across all Tacito Square components.
// A tenant is identified by a domain-syntax TenantID and an optional SubscriptionID.
// The full tenant name is the hyphenated concatenation of both when SubscriptionID is present,
// or TenantID alone when it is absent.
package tenant

import (
	"context"
	"errors"
	"regexp"
	"strings"
)

// domainRe matches a full domain name: one or more dot-separated DNS labels.
// Each label starts and ends with an alphanumeric character; internal hyphens are allowed.
// Examples: "acme.com", "my-org.io", "sub.acme.co.uk", "localhost".
var domainRe = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`)

// subscriptionRe matches an optional subscription identifier: alphanumeric and hyphens.
var subscriptionRe = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$`)

// Tenant represents the resolved identity of a tenant within the platform.
// TenantID follows domain URL syntax (e.g. "acme.com", "my-org.io").
// SubscriptionID is an optional discriminator within a tenant account.
type Tenant struct {
	TenantID       string
	SubscriptionID string // optional; empty string means no subscription qualifier
}

// New constructs a Tenant, validating both fields.
// subscriptionID may be empty.
func New(tenantID, subscriptionID string) (*Tenant, error) {
	t := &Tenant{TenantID: tenantID, SubscriptionID: subscriptionID}
	if err := t.Validate(); err != nil {
		return nil, err
	}
	return t, nil
}

// Validate checks that TenantID is a valid domain-syntax identifier and,
// when present, that SubscriptionID contains only safe characters.
func (t *Tenant) Validate() error {
	if t.TenantID == "" {
		return errors.New("tenant: tenantId is required")
	}
	if !domainRe.MatchString(t.TenantID) {
		return errors.New("tenant: tenantId must follow domain URL syntax (e.g. acme.com)")
	}
	if t.SubscriptionID != "" && !subscriptionRe.MatchString(t.SubscriptionID) {
		return errors.New("tenant: subscriptionId must be alphanumeric with internal hyphens only")
	}
	return nil
}

// FullName returns the canonical tenant name used as a data-partition key.
// When SubscriptionID is present the form is "tenantId-subscriptionId";
// otherwise it is "tenantId" alone.
func (t *Tenant) FullName() string {
	if t.SubscriptionID == "" {
		return t.TenantID
	}
	return strings.Join([]string{t.TenantID, t.SubscriptionID}, "-")
}

// --- Context propagation -------------------------------------------------

type contextKey string

const tenantKey contextKey = "tenant"

// ContextWithTenant returns a derived context carrying the resolved Tenant.
func ContextWithTenant(ctx context.Context, t *Tenant) context.Context {
	return context.WithValue(ctx, tenantKey, t)
}

// FromContext extracts the Tenant from ctx.
// Returns nil if no Tenant has been stored.
func FromContext(ctx context.Context) *Tenant {
	t, _ := ctx.Value(tenantKey).(*Tenant)
	return t
}
