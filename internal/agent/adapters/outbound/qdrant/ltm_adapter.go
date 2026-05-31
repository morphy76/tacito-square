package qdrant

import (
	"context"
	"fmt"
	"time"

	qdrant "github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/morphy76/tacito-square/internal/agent/domain/model"
	"github.com/morphy76/tacito-square/internal/shared/observability"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// QdrantLTMAdapter implements the outbound LongTermMemory interface using Qdrant.
type QdrantLTMAdapter struct {
	conn           *grpc.ClientConn
	pointsClient   qdrant.PointsClient
	collectionName string
	vectorDim      uint64
	tracer         trace.Tracer
}

// NewQdrantLTMAdapter creates a new QdrantLTMAdapter instance.
func NewQdrantLTMAdapter(qdrantURL string, collectionName string, vectorDim uint64) (*QdrantLTMAdapter, error) {
	// 1. Establish gRPC connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, qdrantURL,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to qdrant gRPC: %w", err)
	}

	collectionsClient := qdrant.NewCollectionsClient(conn)
	pointsClient := qdrant.NewPointsClient(conn)

	// 2. Ensure collection exists
	_, err = collectionsClient.Get(ctx, &qdrant.GetCollectionInfoRequest{
		CollectionName: collectionName,
	})
	if err != nil {
		// If collection does not exist, create it
		_, err = collectionsClient.Create(ctx, &qdrant.CreateCollection{
			CollectionName: collectionName,
			VectorsConfig: &qdrant.VectorsConfig{
				Config: &qdrant.VectorsConfig_Params{
					Params: &qdrant.VectorParams{
						Size:     vectorDim,
						Distance: qdrant.Distance_Cosine,
					},
				},
			},
		})
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to create collection %q in qdrant: %w", collectionName, err)
		}
	}

	return &QdrantLTMAdapter{
		conn:           conn,
		pointsClient:   pointsClient,
		collectionName: collectionName,
		vectorDim:      vectorDim,
		tracer:         otel.Tracer("qdrant"),
	}, nil
}

// Close closes the underlying gRPC client connection.
func (a *QdrantLTMAdapter) Close() error {
	if a.conn != nil {
		return a.conn.Close()
	}
	return nil
}

func stringVal(s string) *qdrant.Value {
	return &qdrant.Value{
		Kind: &qdrant.Value_StringValue{StringValue: s},
	}
}

func boolPtr(b bool) *bool {
	return &b
}

// Save stores semantic memory entries under a strictly tenant-isolated scope.
func (a *QdrantLTMAdapter) Save(ctx context.Context, tenantID, agentID string, entries []model.LTMEntry) error {
	start := time.Now()

	ctx, span := a.tracer.Start(ctx, "qdrant.save",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("db.system", "qdrant"),
			attribute.String("db.operation", "upsert"),
			attribute.String("tenant_id", tenantID),
			attribute.String("agent_id", agentID),
			attribute.Int("entries_count", len(entries)),
		),
	)
	defer span.End()

	points := make([]*qdrant.PointStruct, 0, len(entries))
	for _, entry := range entries {
		if err := entry.Validate(); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return fmt.Errorf("invalid memory entry: %w", err)
		}

		visibility := "private"
		if val, ok := entry.Metadata["visibility"]; ok && val != "" {
			visibility = val
		}

		payload := map[string]*qdrant.Value{
			"tenant_id":  stringVal(tenantID),
			"agent_id":   stringVal(agentID),
			"visibility": stringVal(visibility),
			"type":       stringVal(string(entry.Type)),
			"content":    stringVal(entry.Content),
			"source":     stringVal(entry.Source),
			"timestamp":  stringVal(entry.Timestamp.Format(time.RFC3339)),
		}

		if communityID, ok := entry.Metadata["community_id"]; ok && communityID != "" {
			payload["community_id"] = stringVal(communityID)
		}
		if threadID, ok := entry.Metadata["thread_id"]; ok && threadID != "" {
			payload["thread_id"] = stringVal(threadID)
		}

		// Inject any other custom metadata
		for k, v := range entry.Metadata {
			if k != "visibility" && k != "community_id" && k != "thread_id" {
				payload[k] = stringVal(v)
			}
		}

		points = append(points, &qdrant.PointStruct{
			Id:      qdrant.NewID(entry.ID),
			Vectors: qdrant.NewVectorsDense(entry.Embedding),
			Payload: payload,
		})
	}

	_, err := a.pointsClient.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: a.collectionName,
		Wait:           boolPtr(true),
		Points:         points,
	})

	duration := time.Since(start).Seconds()
	status := "success"
	if err != nil {
		status = "failure"
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}

	observability.OutboundDependencyDuration.Record(ctx, duration,
		otelmetric.WithAttributes(
			attribute.String("dependency", "qdrant"),
			attribute.String("operation", "save"),
			attribute.String("status", status),
		),
	)

	if err != nil {
		return fmt.Errorf("qdrant upsert failed: %w", err)
	}
	return nil
}

// Search queries Qdrant for similar memories using the provided vector.
func (a *QdrantLTMAdapter) Search(ctx context.Context, tenantID, agentID string, vector []float32, filter model.LTMFilter, limit int, threshold float32) ([]model.LTMEntry, error) {
	start := time.Now()

	ctx, span := a.tracer.Start(ctx, "qdrant.search",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("db.system", "qdrant"),
			attribute.String("db.operation", "search"),
			attribute.String("tenant_id", tenantID),
			attribute.String("agent_id", agentID),
			attribute.Int("limit", limit),
			attribute.Float64("threshold", float64(threshold)),
		),
	)
	defer span.End()

	qdrantFilter := buildSearchFilter(tenantID, agentID, filter)

	req := &qdrant.SearchPoints{
		CollectionName: a.collectionName,
		Vector:         vector,
		Filter:         qdrantFilter,
		Limit:          uint64(limit),
		WithPayload:    qdrant.NewWithPayload(true),
	}

	if threshold > 0 {
		req.ScoreThreshold = &threshold
	}

	resp, err := a.pointsClient.Search(ctx, req)

	duration := time.Since(start).Seconds()
	status := "success"
	if err != nil {
		status = "failure"
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}

	observability.OutboundDependencyDuration.Record(ctx, duration,
		otelmetric.WithAttributes(
			attribute.String("dependency", "qdrant"),
			attribute.String("operation", "search"),
			attribute.String("status", status),
		),
	)

	if err != nil {
		return nil, fmt.Errorf("qdrant search failed: %w", err)
	}

	entries := make([]model.LTMEntry, 0, len(resp.GetResult()))
	for _, result := range resp.GetResult() {
		payload := result.GetPayload()

		// Helper to extract string from payload
		getString := func(key string) string {
			if val, ok := payload[key]; ok {
				return val.GetStringValue()
			}
			return ""
		}

		timestamp, _ := time.Parse(time.RFC3339, getString("timestamp"))

		metadata := make(map[string]string)
		for k, v := range payload {
			if k != "tenant_id" && k != "agent_id" && k != "type" && k != "content" && k != "source" && k != "timestamp" {
				metadata[k] = v.GetStringValue()
			}
		}

		entries = append(entries, model.LTMEntry{
			ID:        result.GetId().GetUuid(),
			Content:   getString("content"),
			Type:      model.LTMEntryType(getString("type")),
			Source:    getString("source"),
			Timestamp: timestamp,
			Metadata:  metadata,
			Score:     result.GetScore(),
		})
	}

	return entries, nil
}

// Delete removes memories matching specific filters.
func (a *QdrantLTMAdapter) Delete(ctx context.Context, tenantID, agentID string, filter model.LTMFilter) error {
	start := time.Now()

	ctx, span := a.tracer.Start(ctx, "qdrant.delete",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("db.system", "qdrant"),
			attribute.String("db.operation", "delete"),
			attribute.String("tenant_id", tenantID),
			attribute.String("agent_id", agentID),
		),
	)
	defer span.End()

	qdrantFilter := buildSearchFilter(tenantID, agentID, filter)

	_, err := a.pointsClient.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: a.collectionName,
		Points:         qdrant.NewPointsSelectorFilter(qdrantFilter),
		Wait:           boolPtr(true),
	})

	duration := time.Since(start).Seconds()
	status := "success"
	if err != nil {
		status = "failure"
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}

	observability.OutboundDependencyDuration.Record(ctx, duration,
		otelmetric.WithAttributes(
			attribute.String("dependency", "qdrant"),
			attribute.String("operation", "delete"),
			attribute.String("status", status),
		),
	)

	if err != nil {
		return fmt.Errorf("qdrant delete failed: %w", err)
	}
	return nil
}

func buildSearchFilter(tenantID, agentID string, filter model.LTMFilter) *qdrant.Filter {
	// 1. Strict Tenant ID condition
	tenantCondition := qdrant.NewMatch("tenant_id", tenantID)

	// 2. Access control and visibility (Private, Community, and Tenant-shared)
	var visibilityConditions []*qdrant.Condition

	// Private condition: agent_id == agentID AND visibility == "private"
	privateCond := qdrant.NewFilterAsCondition(&qdrant.Filter{
		Must: []*qdrant.Condition{
			qdrant.NewMatch("agent_id", agentID),
			qdrant.NewMatch("visibility", "private"),
		},
	})
	visibilityConditions = append(visibilityConditions, privateCond)

	// Community condition: community_id == communityID AND visibility == "community"
	communityID := filter.CommunityID
	if communityID != "" {
		communityCond := qdrant.NewFilterAsCondition(&qdrant.Filter{
			Must: []*qdrant.Condition{
				qdrant.NewMatch("community_id", communityID),
				qdrant.NewMatch("visibility", "community"),
			},
		})
		visibilityConditions = append(visibilityConditions, communityCond)
	}

	// Tenant condition: visibility == "tenant"
	tenantSharedCond := qdrant.NewMatch("visibility", "tenant")
	visibilityConditions = append(visibilityConditions, tenantSharedCond)

	// Combine tenant ID with visibility conditions
	mustConditions := []*qdrant.Condition{tenantCondition}

	// 3. Types filter (optional)
	if len(filter.Types) > 0 {
		var typeConditions []*qdrant.Condition
		for _, t := range filter.Types {
			typeConditions = append(typeConditions, qdrant.NewMatch("type", string(t)))
		}
		// Any of the types must match
		mustConditions = append(mustConditions, qdrant.NewFilterAsCondition(&qdrant.Filter{
			Should: typeConditions,
		}))
	}

	// 4. ThreadID filter (optional)
	if filter.ThreadID != "" {
		mustConditions = append(mustConditions, qdrant.NewMatch("thread_id", filter.ThreadID))
	}

	return &qdrant.Filter{
		Must:   mustConditions,
		Should: visibilityConditions,
	}
}
