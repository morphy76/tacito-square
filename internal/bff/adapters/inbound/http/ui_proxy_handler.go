package http

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morphy76/tacito-square/internal/bff/application/ports/outbound"
)

// UIProxyHandler proxies index.html requests to the internal Nginx pod and caches the result.
type UIProxyHandler struct {
	store             outbound.SessionStore
	uiConfiguratorURL string
	httpClient        *http.Client
}

// NewUIProxyHandler constructs a UIProxyHandler.
func NewUIProxyHandler(store outbound.SessionStore, uiConfiguratorURL string) *UIProxyHandler {
	if uiConfiguratorURL == "" {
		uiConfiguratorURL = "http://ts-ui-configurator"
	}
	return &UIProxyHandler{
		store:             store,
		uiConfiguratorURL: uiConfiguratorURL,
		httpClient: &http.Client{
			Timeout: 2 * time.Second,
		},
	}
}

// ServeUIIndex handles index.html serving by checking cache first, fetching from Nginx on miss, and caching.
func (h *UIProxyHandler) ServeUIIndex(c *gin.Context) {
	ctx := c.Request.Context()

	// 0. If this is a request for a static asset, proxy it directly
	path := c.Request.URL.Path
	if idx := strings.Index(path, "/assets/"); idx != -1 {
		targetURL := strings.TrimSuffix(h.uiConfiguratorURL, "/") + path[idx:]
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to build request"})
			return
		}

		resp, err := h.httpClient.Do(req)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "UI configurator service unreachable"})
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			c.Status(resp.StatusCode)
			_, _ = io.Copy(c.Writer, resp.Body)
			return
		}

		// Copy headers
		for k, vv := range resp.Header {
			for _, v := range vv {
				c.Header(k, v)
			}
		}
		c.Status(resp.StatusCode)
		_, _ = io.Copy(c.Writer, resp.Body)
		return
	}

	const cacheKey = "index_html"

	// 1. Read from Redis Cache
	html, err := h.store.GetCachedHTML(ctx, cacheKey)
	if err == nil && html != "" {
		c.Header("Cache-Control", "no-cache")
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, html)
		return
	}

	// 2. Cache Miss: Fetch from Nginx
	targetURL := strings.TrimSuffix(h.uiConfiguratorURL, "/") + "/index.html"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to build request"})
		return
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "UI configurator service unreachable"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("UI configurator returned status %d", resp.StatusCode)})
		return
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read response body"})
		return
	}

	htmlContent := string(bodyBytes)

	// 3. Cache the fetched HTML content with a 5-minute TTL
	_ = h.store.CacheHTML(ctx, cacheKey, htmlContent, 5*time.Minute)

	c.Header("Cache-Control", "no-cache")
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, htmlContent)
}
