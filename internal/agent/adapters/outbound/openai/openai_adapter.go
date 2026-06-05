package openai

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/morphy76/tacito-square/internal/agent/adapters/outbound/resiliency"
	"github.com/morphy76/tacito-square/internal/agent/domain/model"
	"github.com/morphy76/tacito-square/internal/shared/observability"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/shared"
	"github.com/rs/zerolog"
	"github.com/sethvargo/go-retry"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
)

type Config struct {
	Endpoint         string
	APIKey           string
	Model            string
	Temperature      float64
	MaxTokens        int
	Timeout          time.Duration
	FailureThreshold int
	RecoveryTimeout  time.Duration
	FallbackMessage  string
	HTTPClient       *http.Client
	AgentName        string
}

type Adapter struct {
	cfg        Config
	client     openai.Client
	cb         *resiliency.CircuitBreaker
	embedderCB *resiliency.CircuitBreaker
}

func NewAdapter(cfg Config) *Adapter {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.FailureThreshold <= 0 {
		cfg.FailureThreshold = 5
	}
	if cfg.RecoveryTimeout <= 0 {
		cfg.RecoveryTimeout = 15 * time.Second
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		}
	}

	opts := []option.RequestOption{
		option.WithAPIKey(cfg.APIKey),
		option.WithHTTPClient(httpClient),
		option.WithMaxRetries(0),
	}
	if cfg.Endpoint != "" {
		opts = append(opts, option.WithBaseURL(cfg.Endpoint))
	}

	client := openai.NewClient(opts...)
	cb := resiliency.NewCircuitBreaker(cfg.FailureThreshold, cfg.RecoveryTimeout)
	embedderCB := resiliency.NewCircuitBreaker(cfg.FailureThreshold, cfg.RecoveryTimeout)

	return &Adapter{
		cfg:        cfg,
		client:     client,
		cb:         cb,
		embedderCB: embedderCB,
	}
}

func (a *Adapter) Generate(ctx context.Context, req model.BrainRequest) (*model.BrainResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Debug().
		Str("model", a.cfg.Model).
		Str("endpoint", a.cfg.Endpoint).
		Msg("entering Generate: sending chat completion request to OpenAI")

	if err := req.Validate(); err != nil {
		logger.Error().Err(err).Msg("invalid brain request payload")
		return nil, err
	}

	var result *model.BrainResponse

	// Define standard operation
	op := func() error {
		runCtx, cancel := context.WithTimeout(ctx, a.cfg.Timeout)
		defer cancel()

		var messages []openai.ChatCompletionMessageParamUnion
		if req.SystemPrompt != "" {
			messages = append(messages, openai.SystemMessage(req.SystemPrompt))
		}
		for _, entry := range req.History {
			switch entry.Role {
			case "assistant":
				messages = append(messages, openai.AssistantMessage(entry.Content))
			case "tool":
				logger.Debug().Str("role", entry.Role).Msg("TODO: tool role history mapping not yet implemented for OpenAI, falling back to user message")
				messages = append(messages, openai.UserMessage(entry.Content))
			default: // "user"
				messages = append(messages, openai.UserMessage(entry.Content))
			}
		}
		messages = append(messages, openai.UserMessage(req.Prompt))

		modelName := a.cfg.Model
		if modelName == "" {
			modelName = "gpt-4o"
		}

		logger.Trace().Interface("messages", messages).Msg("messages for chat completion")
		params := openai.ChatCompletionNewParams{
			Messages: messages,
			Model:    shared.ChatModel(modelName),
		}

		temp := a.cfg.Temperature
		if req.Temperature > 0 {
			temp = req.Temperature
		}
		if temp > 0 {
			params.Temperature = openai.Float(temp)
		}

		maxToks := a.cfg.MaxTokens
		if req.MaxTokens > 0 {
			maxToks = req.MaxTokens
		}
		if maxToks > 0 {
			params.MaxTokens = openai.Int(int64(maxToks))
		}

		// Exponential backoff with jitter retry wrapper using sethvargo/go-retry
		b := retry.NewExponential(10 * time.Millisecond)
		b = retry.WithMaxRetries(3, b)

		logger.Trace().Msg("initiating chat completion wire call (with backoff retry)")
		var chatComp *openai.ChatCompletion
		start := time.Now()
		err := retry.Do(runCtx, b, func(ctx context.Context) error {
			var err error
			chatComp, err = a.client.Chat.Completions.New(ctx, params)
			if err != nil {
				var openaiErr *openai.Error
				if errors.As(err, &openaiErr) && openaiErr.StatusCode == http.StatusTooManyRequests {
					return err
				}
				logger.Warn().Err(err).Msg("chat completion call failed, retrying...")
				return retry.RetryableError(err)
			}
			return nil
		})
		duration := time.Since(start)

		if err != nil {
			logger.Error().Err(err).Dur("duration_ms", duration).Msg("all chat completion retries exhausted")
			brainAttrsWithStatus := otelmetric.WithAttributes(
				attribute.String("agent", a.cfg.AgentName),
				attribute.String("provider", "openai"),
				attribute.String("model", modelName),
				attribute.String("status", "error"),
			)
			observability.AgentBrainRequestsTotal.Add(ctx, 1, brainAttrsWithStatus)
			return err
		}

		if len(chatComp.Choices) == 0 {
			return errors.New("empty choices from openai response")
		}

		logger.Debug().
			Dur("duration_ms", duration).
			Int("prompt_tokens", int(chatComp.Usage.PromptTokens)).
			Int("completion_tokens", int(chatComp.Usage.CompletionTokens)).
			Msg("chat completion wire call completed successfully")

		// Record agent brain metrics
		brainAttrs := otelmetric.WithAttributes(
			attribute.String("agent", a.cfg.AgentName),
			attribute.String("provider", "openai"),
			attribute.String("model", modelName),
		)
		brainAttrsWithStatus := otelmetric.WithAttributes(
			attribute.String("agent", a.cfg.AgentName),
			attribute.String("provider", "openai"),
			attribute.String("model", modelName),
			attribute.String("status", "success"),
		)
		observability.AgentBrainRequestsTotal.Add(ctx, 1, brainAttrsWithStatus)
		observability.AgentBrainRequestDuration.Record(ctx, duration.Seconds(), brainAttrs)

		observability.AgentBrainTokensTotal.Add(ctx, int64(chatComp.Usage.PromptTokens),
			otelmetric.WithAttributes(
				attribute.String("agent", a.cfg.AgentName),
				attribute.String("direction", "sent"),
				attribute.String("model", modelName),
			),
		)
		observability.AgentBrainTokensTotal.Add(ctx, int64(chatComp.Usage.CompletionTokens),
			otelmetric.WithAttributes(
				attribute.String("agent", a.cfg.AgentName),
				attribute.String("direction", "received"),
				attribute.String("model", modelName),
			),
		)

		result = &model.BrainResponse{
			Content: chatComp.Choices[0].Message.Content,
			Usage: model.TokenUsage{
				PromptTokens:     int(chatComp.Usage.PromptTokens),
				CompletionTokens: int(chatComp.Usage.CompletionTokens),
				TotalTokens:      int(chatComp.Usage.TotalTokens),
			},
			FinishReason: string(chatComp.Choices[0].FinishReason),
		}

		return nil
	}

	// Define fallback operation
	fb := func(err error) error {
		logger.Warn().Err(err).Msg("circuit breaker tripped, executing fallback operation")
		result = &model.BrainResponse{
			Content:      a.cfg.FallbackMessage,
			FinishReason: "fallback",
		}
		return nil
	}

	err := a.cb.Execute(ctx, op, fb)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (a *Adapter) GenerateStream(ctx context.Context, req model.BrainRequest) (<-chan model.BrainStreamChunk, <-chan error, error) {
	// Stub streaming logic for milestone baseline
	ch := make(chan model.BrainStreamChunk, 1)
	errCh := make(chan error, 1)

	resp, err := a.Generate(ctx, req)
	if err != nil {
		errCh <- err
		close(ch)
		close(errCh)
		return ch, errCh, err
	}

	ch <- model.BrainStreamChunk{
		Content:      resp.Content,
		FinishReason: resp.FinishReason,
	}
	close(ch)
	close(errCh)
	return ch, errCh, nil
}

// CreateEmbedding generates a high-dimensional dense vector for the given text.
func (a *Adapter) CreateEmbedding(ctx context.Context, text string) ([]float32, error) {
	logger := zerolog.Ctx(ctx)
	logger.Debug().Str("model", a.cfg.Model).Msg("entering CreateEmbedding: generating text embedding")

	var result []float32

	op := func() error {
		runCtx, cancel := context.WithTimeout(ctx, a.cfg.Timeout)
		defer cancel()

		modelName := a.cfg.Model
		if modelName == "" {
			modelName = "text-embedding-3-small"
		}

		params := openai.EmbeddingNewParams{
			Input: openai.EmbeddingNewParamsInputUnion{
				OfString: openai.String(text),
			},
			Model: openai.EmbeddingModel(modelName),
		}

		b := retry.NewExponential(10 * time.Millisecond)
		b = retry.WithMaxRetries(3, b)

		var resp *openai.CreateEmbeddingResponse
		err := retry.Do(runCtx, b, func(ctx context.Context) error {
			var err error
			resp, err = a.client.Embeddings.New(ctx, params)
			if err != nil {
				var openaiErr *openai.Error
				if errors.As(err, &openaiErr) && openaiErr.StatusCode == http.StatusTooManyRequests {
					return err
				}
				logger.Warn().Err(err).Msg("embedding call failed, retrying...")
				return retry.RetryableError(err)
			}
			return nil
		})
		if err != nil {
			logger.Error().Err(err).Msg("all embedding retries exhausted")
			return err
		}

		if len(resp.Data) == 0 {
			return errors.New("empty data from embedding response")
		}

		result = make([]float32, len(resp.Data[0].Embedding))
		for i, v := range resp.Data[0].Embedding {
			result[i] = float32(v)
		}
		return nil
	}

	fb := func(err error) error {
		logger.Warn().Err(err).Msg("embedder circuit breaker tripped")
		return err
	}

	err := a.embedderCB.Execute(ctx, op, fb)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// CreateEmbeddingsBatch generates dense vectors for a slice of texts in parallel.
func (a *Adapter) CreateEmbeddingsBatch(ctx context.Context, texts []string) ([][]float32, error) {
	logger := zerolog.Ctx(ctx)
	logger.Debug().Str("model", a.cfg.Model).Int("batch_size", len(texts)).Msg("entering CreateEmbeddingsBatch: generating batch text embeddings")

	var result [][]float32

	op := func() error {
		runCtx, cancel := context.WithTimeout(ctx, a.cfg.Timeout)
		defer cancel()

		modelName := a.cfg.Model
		if modelName == "" {
			modelName = "text-embedding-3-small"
		}

		params := openai.EmbeddingNewParams{
			Input: openai.EmbeddingNewParamsInputUnion{
				OfArrayOfStrings: texts,
			},
			Model: openai.EmbeddingModel(modelName),
		}

		b := retry.NewExponential(10 * time.Millisecond)
		b = retry.WithMaxRetries(3, b)

		var resp *openai.CreateEmbeddingResponse
		err := retry.Do(runCtx, b, func(ctx context.Context) error {
			var err error
			resp, err = a.client.Embeddings.New(ctx, params)
			if err != nil {
				var openaiErr *openai.Error
				if errors.As(err, &openaiErr) && openaiErr.StatusCode == http.StatusTooManyRequests {
					return err
				}
				logger.Warn().Err(err).Msg("batch embedding call failed, retrying...")
				return retry.RetryableError(err)
			}
			return nil
		})
		if err != nil {
			logger.Error().Err(err).Msg("all batch embedding retries exhausted")
			return err
		}

		result = make([][]float32, len(resp.Data))
		for i, d := range resp.Data {
			vec := make([]float32, len(d.Embedding))
			for j, v := range d.Embedding {
				vec[j] = float32(v)
			}
			result[i] = vec
		}
		return nil
	}

	fb := func(err error) error {
		logger.Warn().Err(err).Msg("batch embedder circuit breaker tripped")
		return err
	}

	err := a.embedderCB.Execute(ctx, op, fb)
	if err != nil {
		return nil, err
	}
	return result, nil
}
