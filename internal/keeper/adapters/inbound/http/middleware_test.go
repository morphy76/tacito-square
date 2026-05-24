package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morphy76/tacito-square/internal/shared/tenant"
	"github.com/stretchr/testify/assert"
)

func TestTenantResolutionMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Successfully resolve tenant and subscription", func(t *testing.T) {
		resolver := NewHeaderTenantResolver()
		r := gin.New()
		r.Use(TenantResolutionMiddleware(resolver))

		var capturedTenant *tenant.Tenant
		r.GET("/test", func(c *gin.Context) {
			capturedTenant = tenant.FromContext(c.Request.Context())
			c.Status(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Tenant-ID", "acme.com")
		req.Header.Set("X-Subscription-ID", "sub-1")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.NotNil(t, capturedTenant)
		assert.Equal(t, "acme.com", capturedTenant.TenantID)
		assert.Equal(t, "sub-1", capturedTenant.SubscriptionID)
		assert.Equal(t, "acme.com-sub-1", capturedTenant.FullName())
	})

	t.Run("Missing Tenant ID header", func(t *testing.T) {
		resolver := NewHeaderTenantResolver()
		r := gin.New()
		r.Use(TenantResolutionMiddleware(resolver))

		r.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "tenantId is required")
	})

	t.Run("Invalid Tenant ID syntax", func(t *testing.T) {
		resolver := NewHeaderTenantResolver()
		r := gin.New()
		r.Use(TenantResolutionMiddleware(resolver))

		r.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Tenant-ID", "-invalid.domain")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "tenantId must follow domain URL syntax")
	})

	t.Run("Whitespace-only Tenant ID", func(t *testing.T) {
		resolver := NewHeaderTenantResolver()
		r := gin.New()
		r.Use(TenantResolutionMiddleware(resolver))

		r.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Tenant-ID", "   ")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "tenantId must follow domain URL syntax")
	})
}
