package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"io"

	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/agent/application/ports/inbound"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
	"github.com/morphy76/tacito-square/internal/shared/observability"
	"github.com/morphy76/tacito-square/internal/shared/ports/outbound"
	natsclient "github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
)

const echoSubjectFormat = "ts.community.%s.agent.%s"

// EchoSubscriber listens for EchoRequest messages on the agent's community subject
// and replies with a decorated EchoReply containing reasoning completions.
type EchoSubscriber struct {
	nc          *natsclient.Conn
	agentName   string
	communityID string
	tenantID    string
	processor   inbound.MessageProcessor
	blobStore   outbound.BlobStore
	logger      zerolog.Logger
	sub         *natsclient.Subscription
}

// NewEchoSubscriber constructs a new EchoSubscriber. Call Start() to begin listening.
func NewEchoSubscriber(nc *natsclient.Conn, agentName, communityID, tenantID string, processor inbound.MessageProcessor, blobStore outbound.BlobStore, logger zerolog.Logger) *EchoSubscriber {
	return &EchoSubscriber{
		nc:          nc,
		agentName:   agentName,
		communityID: communityID,
		tenantID:    tenantID,
		processor:   processor,
		blobStore:   blobStore,
		logger:      logger,
	}
}

// EchoRequestMeta parses metadata from incoming EchoRequests without unmarshaling the entire large message.
type EchoRequestMeta struct {
	TenantID    string          `json:"tenant_id"`
	CommunityID string          `json:"community_id"`
	Timestamp   string          `json:"timestamp"`
	Message     json.RawMessage `json:"message"`
}

// Start subscribes to the agent's echo subject. Returns an error if subscription fails.
func (s *EchoSubscriber) Start(_ context.Context) error {
	s.logger.Debug().Msg("entering Start")
	subject := fmt.Sprintf(echoSubjectFormat, s.communityID, s.agentName)
	sub, err := s.nc.Subscribe(subject,
		observability.WrapNATSHandler("nats.echo_handler", s.logger, s.handleEcho))
	if err != nil {
		return fmt.Errorf("echo subscriber: subscribe to %s: %w", subject, err)
	}
	s.sub = sub
	s.logger.Info().Str("subject", subject).Msg("echo subscriber started")
	return nil
}

// Stop drains and unsubscribes.
func (s *EchoSubscriber) Stop() error {
	s.logger.Debug().Msg("entering Stop")
	if s.sub != nil {
		err := s.sub.Drain()
		s.sub = nil
		return err
	}
	return nil
}

func (s *EchoSubscriber) handleEcho(ctx context.Context, logger zerolog.Logger, msg *natsclient.Msg) error {
	start := time.Now()
	var meta EchoRequestMeta
	if err := json.Unmarshal(msg.Data, &meta); err != nil {
		logger.Warn().Err(err).Msg("echo subscriber: malformed payload, ignoring")
		return nil // intentional: malformed messages are silently ignored, no reply sent
	}

	// Resolve thread ID from headers or fallback to community-scoped thread for hello-world session continuity
	threadID := msg.Header.Get("X-Thread-ID")
	if threadID == "" {
		threadID = msg.Header.Get("thread_id")
	}
	if threadID == "" {
		threadID = "thread-" + s.communityID
	}

	rawMsg := meta.Message
	if len(rawMsg) >= 2 && rawMsg[0] == '"' && rawMsg[len(rawMsg)-1] == '"' {
		rawMsg = rawMsg[1 : len(rawMsg)-1]
	}

	var messageText string
	bucketName := NormalizeBucketName(meta.TenantID)
	if bucketName == "" {
		bucketName = "default"
	}

	// Trigger offload if size is > 256KB and blobStore is configured
	if len(rawMsg) > 256*1024 && s.blobStore != nil {
		objectID := uuid.New().String()
		s3Key := fmt.Sprintf("%s/ingress/%s/%s/%s", s.communityID, s.agentName, threadID, objectID)

		// Upload stream-buffered directly from NATS message body reader
		reader := &unescapeReader{data: rawMsg}
		_, err := s.blobStore.Put(ctx, s3Key, reader, "text/plain")
		if err != nil {
			logger.Error().Err(err).Str("key", s3Key).Msg("echo subscriber: failed to offload payload to S3")
			return fmt.Errorf("offload payload to S3: %w", err)
		}

		ref := S3Reference{
			Type:        "s3_reference",
			Bucket:      bucketName,
			Key:         s3Key,
			SizeBytes:   int64(len(rawMsg)),
			ContentType: "text/plain",
		}
		refBytes, err := json.Marshal(ref)
		if err != nil {
			logger.Error().Err(err).Msg("echo subscriber: failed to marshal s3_reference")
			return fmt.Errorf("marshal s3_reference: %w", err)
		}
		messageText = string(refBytes)

		logger.Info().
			Str("agent_name", s.agentName).
			Str("community_id", s.communityID).
			Str("tenant_id", meta.TenantID).
			Str("bucket", bucketName).
			Str("key", s3Key).
			Int64("size_bytes", ref.SizeBytes).
			Msg("echo request payload offloaded to object storage")
	} else {
		// Small message
		reader := &unescapeReader{data: rawMsg}
		decoded, err := io.ReadAll(reader)
		if err != nil {
			logger.Warn().Err(err).Msg("echo subscriber: failed to decode message")
			return fmt.Errorf("decode message: %w", err)
		}
		messageText = string(decoded)

		sanitized := model.SanitizeMessage(messageText)
		logger.Debug().
			Str("agent_name", s.agentName).
			Str("community_id", s.communityID).
			Str("tenant_id", meta.TenantID).
			Str("message", sanitized).
			Msg("echo request received")
	}

	// Store enriched logger in context so downstream reasoning pipeline retains trace correlation
	ctx = logger.WithContext(ctx)

	// Trigger the message processing framework pipeline (Brain reasoning engine)
	brainResult, err := s.processor.ProcessIncomingMessage(ctx, meta.TenantID, s.agentName, threadID, messageText)

	// Record metrics
	duration := time.Since(start).Seconds()
	statusAttr := "success"
	if err != nil {
		statusAttr = "error"
	}
	subject := msg.Subject

	attrs := otelmetric.WithAttributes(
		attribute.String("agent", s.agentName),
		attribute.String("community", s.communityID),
		attribute.String("subject", subject),
	)
	attrsWithStatus := otelmetric.WithAttributes(
		attribute.String("agent", s.agentName),
		attribute.String("community", s.communityID),
		attribute.String("subject", subject),
		attribute.String("status", statusAttr),
	)

	observability.AgentNATSMessagesProcessedTotal.Add(ctx, 1, attrsWithStatus)
	observability.AgentNATSProcessingDuration.Record(ctx, duration, attrs)

	if err != nil {
		logger.Warn().Err(err).Msg("echo subscriber: message processing failed")
		return fmt.Errorf("process incoming message: %w", err)
	}

	decorated := model.DecorateMessage(s.agentName, brainResult)
	now := time.Now().UTC()

	reply := model.EchoReply{
		AgentName: s.agentName,
		Decorated: decorated,
		Timestamp: now.Format(time.RFC3339),
	}

	data, err := json.Marshal(reply)
	if err != nil {
		logger.Warn().Err(err).Msg("echo subscriber: failed to marshal reply")
		return fmt.Errorf("marshal echo reply: %w", err)
	}

	if err := msg.Respond(data); err != nil {
		logger.Warn().Err(err).Msg("echo subscriber: failed to send reply")
		return fmt.Errorf("respond to echo request: %w", err)
	}

	return nil
}
