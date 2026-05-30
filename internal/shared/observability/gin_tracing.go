package observability

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/morphy76/tacito-square/internal/shared/tenant"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// SpanEventFromGinContext emits a named event on the span currently active in c's
// request context, automatically enriching it with tenant_id (if available) and
// any caller-supplied attributes.
//
// Handlers should call this once per significant domain milestone, e.g.:
//
//	observability.SpanEventFromGinContext(c, "keeper.request.accepted",
//	    attribute.String("community_id", commID.String()),
//	)
//
// This keeps all OTel event emission out of the handler body and in the framework layer.
func SpanEventFromGinContext(c *gin.Context, name string, attrs ...attribute.KeyValue) {
	ctx := c.Request.Context()
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return
	}
	all := ginContextAttrs(ctx, attrs...)
	span.AddEvent(name, trace.WithAttributes(all...))
}

// RecordGinError stamps the active span in c with a structured error event and
// sets the span status to Error. Call this at every handler error return path
// where the framework middleware cannot infer the domain reason (e.g., 4xx
// business-logic failures that the TracingMiddleware doesn't treat as errors).
//
// Example:
//
//	observability.RecordGinError(c, http.StatusNotFound, err,
//	    attribute.String("community_id", commID.String()),
//	)
func RecordGinError(c *gin.Context, statusCode int, err error, attrs ...attribute.KeyValue) {
	ctx := c.Request.Context()
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return
	}
	all := ginContextAttrs(ctx,
		append([]attribute.KeyValue{
			attribute.Int("http.status_code", statusCode),
			attribute.String("error.message", err.Error()),
		}, attrs...)...,
	)
	span.RecordError(err)
	span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d: %s", statusCode, err.Error()))
	span.AddEvent("http.error", trace.WithAttributes(all...))
}

// StartHandlerSpan reuses and renames the active Gin server span to operationName,
// avoiding invalid nested server spans. It returns the updated context carrying the span
// and the span itself.
func StartHandlerSpan(c *gin.Context, operationName string) (context.Context, trace.Span) {
	ctx := c.Request.Context()
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		span.SetName(operationName)
	}
	return ctx, span
}

// ginContextAttrs builds the standard event attribute set for a Gin handler span
// event: tenant_id (from context if available) plus any caller-supplied extras.
func ginContextAttrs(ctx context.Context, extra ...attribute.KeyValue) []attribute.KeyValue {
	attrs := make([]attribute.KeyValue, 0, 1+len(extra))
	if t := tenant.FromContext(ctx); t != nil {
		attrs = append(attrs, attribute.String("tenant_id", t.FullName()))
	}
	return append(attrs, extra...)
}

// upgradeTracingMiddleware extends the existing TracingMiddleware post-processing
// to emit span events for both 4xx and 5xx responses, making all error paths
// visible in distributed traces without touching individual handlers.
//
// This is injected into the TracingMiddleware via afterRequest.
func afterRequest(c *gin.Context) {
	status := c.Writer.Status()
	if status < http.StatusBadRequest {
		return
	}
	ctx := c.Request.Context()
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return
	}

	errMsg := c.Errors.Last()
	msg := http.StatusText(status)
	if errMsg != nil {
		msg = errMsg.Error()
	}

	attrs := ginContextAttrs(ctx,
		attribute.Int("http.status_code", status),
		attribute.String("http.method", c.Request.Method),
		attribute.String("http.route", c.FullPath()),
		attribute.String("error.message", msg),
	)

	eventName := "http.client_error"
	if status >= http.StatusInternalServerError {
		eventName = "http.server_error"
		span.RecordError(fmt.Errorf("HTTP %d %s", status, msg))
		span.SetStatus(codes.Error, strconv.Itoa(status))
	}
	span.AddEvent(eventName, trace.WithAttributes(attrs...))
}
