package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/morphy76/tacito-square/internal/agent/application/service"
	"github.com/morphy76/tacito-square/internal/agent/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestReasoningOTel_Instrumentation(t *testing.T) {
	t.Run("happy path: records span events and errors in reasoning loop", func(t *testing.T) {
		// Set up standard OTel in-memory span exporter
		exporter := tracetest.NewInMemoryExporter()
		tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
		otel.SetTracerProvider(tp)

		stepCount := 0
		mockBrain := &MockBrain{
			GenerateFunc: func(ctx context.Context, request model.BrainRequest) (*model.BrainResponse, error) {
				stepCount++
				if stepCount == 1 {
					toolCall := map[string]any{
						"thought": "Deciding to recall memory.",
						"tool_call": map[string]any{
							"name": "recall_memory",
							"arguments": map[string]any{
								"query": "db connection info",
							},
						},
					}
					data, _ := json.Marshal(toolCall)
					return &model.BrainResponse{Content: string(data)}, nil
				}

				finalAnswer := map[string]any{
					"final_answer": "According to memory, connection is verified.",
				}
				data, _ := json.Marshal(finalAnswer)
				return &model.BrainResponse{Content: string(data)}, nil
			},
		}

		engine := service.NewCognitiveEngine(mockBrain, 5, nil)
		engine.RegisterTool("recall_memory", func(ctx context.Context, args map[string]any) (string, error) {
			// Trigger a simulated error in tool to verify RecordError
			return "", errors.New("simulated database timeout")
		})

		ctx := context.Background()
		_, err := engine.ExecuteReasoningLoop(ctx, "tenant-1", "agent-1", "thread-1", "Find database config", []model.MemoryEntry{}, "")
		assert.NoError(t, err)

		// Force trace provider processing
		err = tp.ForceFlush(ctx)
		assert.NoError(t, err)

		spans := exporter.GetSpans()
		require.NotEmpty(t, spans)

		// 1. Assert OTel Span was created and named correctly
		var stepSpans []tracetest.SpanStub
		for _, s := range spans {
			if s.Name == "cognitive_engine.step" {
				stepSpans = append(stepSpans, s)
			}
		}
		require.NotEmpty(t, stepSpans)

		// 2. Assert Span Events are registered for thoughts and tool calls
		eventFound := false
		for _, s := range stepSpans {
			for _, e := range s.Events {
				if e.Name == "thought" {
					eventFound = true
					assert.Contains(t, e.Attributes[0].Value.AsString(), "Deciding to recall memory")
				}
			}
		}
		assert.True(t, eventFound)

		// 3. Assert simulated tool failure records exception on span and sets status to Error
		errorStatusFound := false
		exceptionFound := false
		for _, s := range stepSpans {
			if s.Status.Code == codes.Error {
				errorStatusFound = true
				assert.Contains(t, s.Status.Description, "simulated database timeout")
			}
			for _, e := range s.Events {
				if e.Name == "exception" {
					exceptionFound = true
					// exception message attribute value
					msgAttrFound := false
					for _, attr := range e.Attributes {
						if attr.Key == "exception.message" {
							msgAttrFound = true
							assert.Contains(t, attr.Value.AsString(), "simulated database timeout")
						}
					}
					assert.True(t, msgAttrFound)
				}
			}
		}
		assert.True(t, errorStatusFound)
		assert.True(t, exceptionFound)
	})

	t.Run("safeguard limit: records safeguard_limit_reached event on parent loop span and sets Ok status", func(t *testing.T) {
		exporter := tracetest.NewInMemoryExporter()
		tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
		otel.SetTracerProvider(tp)

		mockBrain := &MockBrain{
			GenerateFunc: func(ctx context.Context, request model.BrainRequest) (*model.BrainResponse, error) {
				// Keep returning tool calls to force hitting the safeguard limit
				toolCall := map[string]any{
					"thought": "Looping forever...",
					"tool_call": map[string]any{
						"name":      "infinite_tool",
						"arguments": map[string]any{},
					},
				}
				data, _ := json.Marshal(toolCall)
				return &model.BrainResponse{Content: string(data)}, nil
			},
		}

		engine := service.NewCognitiveEngine(mockBrain, 3, nil)
		engine.RegisterTool("infinite_tool", func(ctx context.Context, args map[string]any) (string, error) {
			return "some output", nil
		})

		ctx := context.Background()
		_, err := engine.ExecuteReasoningLoop(ctx, "tenant-1", "agent-1", "thread-1", "infinite loop query", []model.MemoryEntry{}, "")
		assert.NoError(t, err)

		err = tp.ForceFlush(ctx)
		assert.NoError(t, err)

		spans := exporter.GetSpans()
		require.NotEmpty(t, spans)

		var parentSpan *tracetest.SpanStub
		for _, s := range spans {
			if s.Name == "cognitive_engine.loop" {
				parentSpan = &s
				break
			}
		}

		require.NotNil(t, parentSpan)

		// 1. Assert safeguard event was added
		eventFound := false
		for _, e := range parentSpan.Events {
			if e.Name == "safeguard_limit_reached" {
				eventFound = true
				maxStepsVal := e.Attributes[0].Value.AsInterface()
				assert.Equal(t, int64(3), maxStepsVal)
				assert.Contains(t, e.Attributes[1].Value.AsString(), "exceeded maximum reasoning steps limit")
			}
		}
		assert.True(t, eventFound)

		// 2. Assert parent span status is Ok (safeguard is NOT an error)
		assert.Equal(t, codes.Ok, parentSpan.Status.Code)
	})
}
