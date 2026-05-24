package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/morphy76/tacito-square/internal/shared/tenant"
)

// TenantResolver defines the contract for resolving the tenant dynamically from an HTTP context.
type TenantResolver interface {
	Resolve(c *gin.Context) (*tenant.Tenant, error)
}

// HeaderTenantResolver resolves tenant identity using custom HTTP headers.
type HeaderTenantResolver struct {
	TenantIDHeader       string
	SubscriptionIDHeader string
}

// NewHeaderTenantResolver creates a new HeaderTenantResolver with default headers.
func NewHeaderTenantResolver() *HeaderTenantResolver {
	return &HeaderTenantResolver{
		TenantIDHeader:       "X-Tenant-ID",
		SubscriptionIDHeader: "X-Subscription-ID",
	}
}

// Resolve extracts and validates tenant details from standard headers.
func (r *HeaderTenantResolver) Resolve(c *gin.Context) (*tenant.Tenant, error) {
	tenantID := c.GetHeader(r.TenantIDHeader)
	subscriptionID := c.GetHeader(r.SubscriptionIDHeader)

	return tenant.New(tenantID, subscriptionID)
}

// TenantResolutionMiddleware returns a Gin middleware using the given resolver to enforce multitenancy bounds.
func TenantResolutionMiddleware(resolver TenantResolver) gin.HandlerFunc {
	return func(c *gin.Context) {
		resolvedTenant, err := resolver.Resolve(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		// Propagate resolved tenant through request context
		ctx := tenant.ContextWithTenant(c.Request.Context(), resolvedTenant)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// DatabaseAvailabilityMiddleware returns a Gin middleware that checks database availability.
// If the database pool is nil, it aborts with 503 Service Unavailable.
func DatabaseAvailabilityMiddleware(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if pool == nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "Database service unavailable"})
			return
		}
		c.Next()
	}
}
