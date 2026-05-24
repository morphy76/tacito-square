package observability

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestMetricsMiddleware_InstrumentsRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(MetricsMiddleware())

	r.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRegisterDBPoolStats_NilPool_NoPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		RegisterDBPoolStats(nil)
	})
}

func TestDBPoolCollector_NilPool_NoPanic(t *testing.T) {
	collector := NewDBPoolCollector(nil)
	descs := make(chan *prometheus.Desc, 10)
	assert.NotPanics(t, func() {
		collector.Describe(descs)
	})

	metrics := make(chan prometheus.Metric, 10)
	assert.NotPanics(t, func() {
		collector.Collect(metrics)
	})
}

func TestCustomPrometheusMetrics_Registration(t *testing.T) {
	assert.NotNil(t, ActiveThreads)
	assert.NotNil(t, AgentStatus)
	assert.NotNil(t, PendingHITLCallbacks)
	assert.NotNil(t, CommunityQuotaUtilization)
	assert.NotNil(t, AgentQuotaUtilization)
	assert.NotNil(t, OutboundDependencyDuration)
}
