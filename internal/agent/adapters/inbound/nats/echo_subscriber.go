package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"io"
	"strings"

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

// NormalizeBucketName normalizes a tenant name to meet S3 bucket requirements.
func NormalizeBucketName(tenantName string) string {
	name := strings.ToLower(tenantName)
	var sb strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			sb.WriteRune(r)
		} else {
			sb.WriteRune('-')
		}
	}
	name = sb.String()
	for strings.Contains(name, "--") {
		name = strings.ReplaceAll(name, "--", "-")
	}
	name = strings.Trim(name, "-")
	if len(name) > 63 {
		name = name[:63]
		name = strings.TrimSuffix(name, "-")
	}
	return name
}

// EchoRequestMeta parses metadata from incoming EchoRequests without unmarshaling the entire large message.
type EchoRequestMeta struct {
	TenantID    string          `json:"tenant_id"`
	CommunityID string          `json:"community_id"`
	Timestamp   string          `json:"timestamp"`
	Message     json.RawMessage `json:"message"`
}

// S3Reference represents the structured JSON offloading payload reference.
type S3Reference struct {
	Type        string `json:"_type"`
	Bucket      string `json:"bucket"`
	Key         string `json:"key"`
	SizeBytes   int64  `json:"size_bytes"`
	ContentType string `json:"content_type"`
}

// unescapeReader decodes JSON string escapes streamingly to avoid flat in-memory slice allocations.
type unescapeReader struct {
	data []byte
	pos  int
}

func (r *unescapeReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = 0
	for r.pos < len(r.data) && n < len(p) {
		c := r.data[r.pos]
		if c == '\\' {
			if r.pos+1 >= len(r.data) {
				p[n] = c
				n++
				r.pos++
				continue
			}
			next := r.data[r.pos+1]
			switch next {
			case 'n':
				p[n] = '\n'
				r.pos += 2
			case 't':
				p[n] = '\t'
				r.pos += 2
			case 'r':
				p[n] = '\r'
				r.pos += 2
			case '\\':
				p[n] = '\\'
				r.pos += 2
			case '"':
				p[n] = '"'
				r.pos += 2
			case '/':
				p[n] = '/'
				r.pos += 2
			case 'u':
				if r.pos+5 < len(r.data) {
					hexVal := 0
					ok := true
					for i := 0; i < 4; i++ {
						h := r.data[r.pos+2+i]
						hexVal <<= 4
						if h >= '0' && h <= '9' {
							hexVal += int(h - '0')
						} else if h >= 'a' && h <= 'f' {
							hexVal += int(h - 'a' + 10)
						} else if h >= 'A' && h <= 'F' {
							hexVal += int(h - 'A' + 10)
						} else {
							ok = false
							break
						}
					}
					if ok {
						p[n] = byte(hexVal)
						r.pos += 6
					} else {
						p[n] = c
						r.pos++
					}
				} else {
					p[n] = c
					r.pos++
				}
			default:
				p[n] = c
				r.pos++
			}
		} else {
			p[n] = c
			r.pos++
		}
		n++
	}
	return n, nil
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
