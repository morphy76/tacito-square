package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/agent/domain/model"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// handleReadLargePayload implements the read_large_payload tool.
func (e *CognitiveEngine) handleReadLargePayload(ctx context.Context, args map[string]any) (string, error) {
	logger := zerolog.Ctx(ctx)

	if e.blobStore == nil {
		logger.Warn().Msg("read_large_payload called but blobStore is nil")
		return `{"error": "Object storage temporarily unavailable."}`, nil
	}

	key, ok := args["key"].(string)
	if !ok || key == "" {
		return "Error: key parameter must be a non-empty string", nil
	}

	// Read from S3 using the shared BlobStore
	reader, err := e.blobStore.Get(ctx, key)
	if err != nil {
		logger.Warn().Err(err).Str("key", key).Msg("failed to read from object storage")
		return `{"error": "Object storage temporarily unavailable."}`, nil
	}
	defer reader.Close()

	// Implement stream-buffered reading from S3 protected by maxReadSize safety limits
	limitedReader := io.LimitReader(reader, int64(e.maxReadSize))

	var sb strings.Builder
	buf := make([]byte, e.chunkSize)
	for {
		nr, er := limitedReader.Read(buf)
		if nr > 0 {
			sb.Write(buf[:nr])
		}
		if er != nil {
			if er == io.EOF {
				break
			}
			logger.Warn().Err(er).Str("key", key).Msg("failed during buffered read from S3")
			return `{"error": "Object storage temporarily unavailable."}`, nil
		}
	}

	return sb.String(), nil
}

// handleWriteLargePayload implements the write_large_payload tool.
func (e *CognitiveEngine) handleWriteLargePayload(ctx context.Context, args map[string]any) (string, error) {
	logger := zerolog.Ctx(ctx)

	if e.blobStore == nil {
		logger.Warn().Msg("write_large_payload called but blobStore is nil")
		return `{"error": "Object storage temporarily unavailable."}`, nil
	}

	content, ok := args["content"].(string)
	if !ok || content == "" {
		return "Error: content parameter must be a non-empty string", nil
	}

	contentType, _ := args["content_type"].(string)
	if contentType == "" {
		contentType = "text/plain"
	}

	tenantID := GetTenantID(ctx)
	agentID := GetAgentID(ctx)
	threadID := GetThreadID(ctx)

	if tenantID == "" {
		tenantID = "default"
	}
	if agentID == "" {
		agentID = "default"
	}
	if threadID == "" {
		threadID = "default"
	}

	normalizedBucket := normalizeBucketName(tenantID)
	objectID := uuid.New().String()

	commID := e.communityID
	if commID == "" {
		commID = "default"
	}

	s3Key := fmt.Sprintf("%s/output/%s/%s/%s", commID, agentID, threadID, objectID)

	reader := strings.NewReader(content)

	ctx, span := e.tracer.Start(ctx, "cognitive_engine.write_large_payload",
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String("s3.bucket", normalizedBucket),
			attribute.String("s3.key", s3Key),
			attribute.Int64("s3.size_bytes", int64(len(content))),
		),
	)
	defer span.End()

	_, err := e.blobStore.Put(ctx, s3Key, reader, contentType)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.Warn().Err(err).Str("key", s3Key).Msg("failed to write to object storage")
		return `{"error": "Object storage temporarily unavailable."}`, nil
	}

	ref := map[string]any{
		"_type":        "s3_reference",
		"bucket":       normalizedBucket,
		"key":          s3Key,
		"size_bytes":   int64(len(content)),
		"content_type": contentType,
	}
	refJSON, _ := json.Marshal(ref)
	return string(refJSON), nil
}

// normalizeBucketName formats the tenant ID to comply with S3 naming conventions.
func normalizeBucketName(tenantName string) string {
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

// handleRecallMemory implements the recall_memory tool.
func (e *CognitiveEngine) handleRecallMemory(ctx context.Context, args map[string]any) (string, error) {
	logger := zerolog.Ctx(ctx)

	bypassLTM := false
	if e.cfg != nil {
		bypassLTM = e.cfg.GetBool("bypass.ltm")
	}
	if bypassLTM {
		logger.Debug().Msg("recall_memory bypassed because bypass.ltm is set to true in config")
		return "No relevant memories found.", nil
	}

	if e.embedder == nil || e.ltm == nil {
		logger.Warn().Msg("recall_memory called but long-term memory or embedder is nil")
		return `{"error": "Memory store temporarily unavailable."}`, nil
	}

	query, ok := args["query"].(string)
	if !ok || query == "" {
		return "Error: query parameter must be a non-empty string", nil
	}

	limit := 3
	if limitVal, ok := args["limit"]; ok {
		switch v := limitVal.(type) {
		case float64:
			limit = int(v)
		case int:
			limit = v
		}
	}

	var filter model.LTMFilter
	if catVal, ok := args["category"].(string); ok && catVal != "" {
		filter.Types = []model.LTMEntryType{model.LTMEntryType(catVal)}
	}

	tenantID := GetTenantID(ctx)
	agentID := GetAgentID(ctx)

	// 1. Generate text embedding vector
	vector, err := e.embedder.CreateEmbedding(ctx, query)
	if err != nil {
		logger.Warn().Err(err).Msg("failed to generate embedding for recall_memory tool")
		return `{"error": "Memory store temporarily unavailable."}`, nil
	}

	// 2. Perform similarity search in Qdrant LTM
	threshold := float32(0.7)
	matches, err := e.ltm.Search(ctx, tenantID, agentID, vector, filter, limit, threshold)
	if err != nil {
		logger.Warn().Err(err).Msg("failed to query Qdrant LTM for recall_memory tool")
		return `{"error": "Memory store temporarily unavailable."}`, nil
	}

	if len(matches) == 0 {
		return "No relevant memories found.", nil
	}

	// 3. Format matches
	var sb strings.Builder
	sb.WriteString("Matched context details:\n")
	for _, match := range matches {
		sb.WriteString(fmt.Sprintf("- %s\n", match.Content))
	}
	return sb.String(), nil
}

// handleEnableSkill implements the enable_skill tool.
func (e *CognitiveEngine) handleEnableSkill(ctx context.Context, args map[string]any) (string, error) {
	logger := zerolog.Ctx(ctx)

	skillName, _ := args["skill_name"].(string)
	if skillName == "" {
		return "Error: skill_name parameter must be a non-empty string", nil
	}

	// Try request-scoped parsed skills first
	var skill Skill
	var exists bool
	if skills, ok := ctx.Value(parsedSkillsKey{}).(map[string]Skill); ok {
		skill, exists = skills[skillName]
	}
	// Fallback to static registered engine skills
	if !exists {
		skill, exists = e.skills[skillName]
	}

	if !exists {
		logger.Warn().Str("skill", skillName).Msg("enable_skill requested for unauthorized or non-existent skill")
		return "Skill unauthorized or not found.", nil
	}

	return fmt.Sprintf("Skill %s enabled successfully. Procedural Guidelines:\n%s", skillName, skill.Content), nil
}
