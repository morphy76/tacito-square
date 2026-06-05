package ollama

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/morphy76/tacito-square/internal/agent/adapters/outbound/resiliency"
	"github.com/morphy76/tacito-square/internal/agent/domain/model"
	"github.com/morphy76/tacito-square/internal/shared/observability"
	"github.com/ollama/ollama/api"
	"github.com/rs/zerolog"
	"github.com/sethvargo/go-retry"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
)

type Config struct {
	Endpoint         string
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
	client     *api.Client
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

	endpointURL, err := url.Parse(cfg.Endpoint)
	if err != nil || cfg.Endpoint == "" {
		endpointURL = &url.URL{Scheme: "http", Host: "127.0.0.1:11434"}
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		}
	}

	client := api.NewClient(endpointURL, httpClient)
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
		Msg("entering Generate: sending chat completion request to Ollama")
	logger.Trace().Interface("request_body", req).Msg("full brain request payload")

	if err := req.Validate(); err != nil {
		logger.Error().Err(err).Msg("invalid brain request payload")
		return nil, err
	}

	var result *model.BrainResponse

	// Define standard operation
	op := func() error {
		runCtx, cancel := context.WithTimeout(ctx, a.cfg.Timeout)
		defer cancel()

		var messages []api.Message
		if req.SystemPrompt != "" {
			messages = append(messages, api.Message{
				Role:    "system",
				Content: req.SystemPrompt,
			})
		}
		messages = append(messages, api.Message{
			Role:    "user",
			Content: req.Prompt,
		})

		modelName := a.cfg.Model
		if modelName == "" {
			modelName = "llama3"
		}

		streamVal := false
		options := make(map[string]interface{})
		temp := a.cfg.Temperature
		if req.Temperature > 0 {
			temp = req.Temperature
		}
		if temp > 0 {
			options["temperature"] = temp
		}

		maxToks := a.cfg.MaxTokens
		if req.MaxTokens > 0 {
			maxToks = req.MaxTokens
		}
		if maxToks > 0 {
			options["num_predict"] = maxToks
		}

		params := &api.ChatRequest{
			Model:    modelName,
			Messages: messages,
			Stream:   &streamVal,
			Options:  options,
		}

		// Exponential backoff with jitter retry wrapper using sethvargo/go-retry
		b := retry.NewExponential(10 * time.Millisecond)
		b = retry.WithMaxRetries(3, b)

		var responseText string
		var promptEvalCount int
		var evalCount int

		logger.Trace().Msg("initiating Ollama chat wire call (with backoff retry)")
		start := time.Now()
		err := retry.Do(runCtx, b, func(ctx context.Context) error {
			var err error
			err = a.client.Chat(ctx, params, func(resp api.ChatResponse) error {
				responseText += resp.Message.Content
				promptEvalCount = resp.PromptEvalCount
				evalCount = resp.EvalCount
				return nil
			})
			if err != nil {
				var statusErr api.StatusError
				if errors.As(err, &statusErr) && statusErr.StatusCode == http.StatusTooManyRequests {
					return err
				}
				logger.Warn().Err(err).Msg("Ollama chat call failed, retrying...")
				return retry.RetryableError(err)
			}
			return nil
		})
		duration := time.Since(start)

		if err != nil {
			logger.Error().Err(err).Dur("duration_ms", duration).Msg("all Ollama chat retries exhausted")
			brainAttrsWithStatus := otelmetric.WithAttributes(
				attribute.String("agent", a.cfg.AgentName),
				attribute.String("provider", "ollama"),
				attribute.String("model", modelName),
				attribute.String("status", "error"),
			)
			observability.AgentBrainRequestsTotal.Add(ctx, 1, brainAttrsWithStatus)
			return err
		}

		logger.Debug().
			Dur("duration_ms", duration).
			Int("prompt_tokens", promptEvalCount).
			Int("completion_tokens", evalCount).
			Msg("Ollama chat wire call completed successfully")

		// Record agent brain metrics
		brainAttrs := otelmetric.WithAttributes(
			attribute.String("agent", a.cfg.AgentName),
			attribute.String("provider", "ollama"),
			attribute.String("model", modelName),
		)
		brainAttrsWithStatus := otelmetric.WithAttributes(
			attribute.String("agent", a.cfg.AgentName),
			attribute.String("provider", "ollama"),
			attribute.String("model", modelName),
			attribute.String("status", "success"),
		)
		observability.AgentBrainRequestsTotal.Add(ctx, 1, brainAttrsWithStatus)
		observability.AgentBrainRequestDuration.Record(ctx, duration.Seconds(), brainAttrs)

		observability.AgentBrainTokensTotal.Add(ctx, int64(promptEvalCount),
			otelmetric.WithAttributes(
				attribute.String("agent", a.cfg.AgentName),
				attribute.String("direction", "sent"),
				attribute.String("model", modelName),
			),
		)
		observability.AgentBrainTokensTotal.Add(ctx, int64(evalCount),
			otelmetric.WithAttributes(
				attribute.String("agent", a.cfg.AgentName),
				attribute.String("direction", "received"),
				attribute.String("model", modelName),
			),
		)

		result = &model.BrainResponse{
			Content: responseText,
			Usage: model.TokenUsage{
				PromptTokens:     promptEvalCount,
				CompletionTokens: evalCount,
				TotalTokens:      promptEvalCount + evalCount,
			},
			FinishReason: "stop",
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
	logger.Debug().Str("model", a.cfg.Model).Msg("entering CreateEmbedding: generating text embedding via Ollama")

	var result []float32

	op := func() error {
		runCtx, cancel := context.WithTimeout(ctx, a.cfg.Timeout)
		defer cancel()

		modelName := a.cfg.Model
		if modelName == "" {
			modelName = "nomic-embed-text"
		}

		req := &api.EmbedRequest{
			Model: modelName,
			Input: text,
		}

		b := retry.NewExponential(10 * time.Millisecond)
		b = retry.WithMaxRetries(3, b)

		var resp *api.EmbedResponse
		err := retry.Do(runCtx, b, func(ctx context.Context) error {
			var err error
			resp, err = a.client.Embed(ctx, req)
			if err != nil {
				var statusErr api.StatusError
				if errors.As(err, &statusErr) && statusErr.StatusCode == http.StatusTooManyRequests {
					return err
				}
				logger.Warn().Err(err).Msg("Ollama embedding call failed, retrying...")
				return retry.RetryableError(err)
			}
			return nil
		})
		if err != nil {
			logger.Error().Err(err).Msg("all Ollama embedding retries exhausted")
			return err
		}

		if len(resp.Embeddings) == 0 {
			return errors.New("empty embeddings from Ollama response")
		}

		result = make([]float32, len(resp.Embeddings[0]))
		for i, v := range resp.Embeddings[0] {
			result[i] = float32(v)
		}
		return nil
	}

	fb := func(err error) error {
		logger.Warn().Err(err).Msg("Ollama embedder circuit breaker tripped")
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
	logger.Debug().Str("model", a.cfg.Model).Int("batch_size", len(texts)).Msg("entering CreateEmbeddingsBatch: generating batch text embeddings via Ollama")

	var result [][]float32

	op := func() error {
		runCtx, cancel := context.WithTimeout(ctx, a.cfg.Timeout)
		defer cancel()

		modelName := a.cfg.Model
		if modelName == "" {
			modelName = "nomic-embed-text"
		}

		req := &api.EmbedRequest{
			Model: modelName,
			Input: texts,
		}

		b := retry.NewExponential(10 * time.Millisecond)
		b = retry.WithMaxRetries(3, b)

		var resp *api.EmbedResponse
		err := retry.Do(runCtx, b, func(ctx context.Context) error {
			var err error
			resp, err = a.client.Embed(ctx, req)
			if err != nil {
				var statusErr api.StatusError
				if errors.As(err, &statusErr) && statusErr.StatusCode == http.StatusTooManyRequests {
					return err
				}
				logger.Warn().Err(err).Msg("Ollama batch embedding call failed, retrying...")
				return retry.RetryableError(err)
			}
			return nil
		})
		if err != nil {
			logger.Error().Err(err).Msg("all Ollama batch embedding retries exhausted")
			return err
		}

		result = make([][]float32, len(resp.Embeddings))
		for i, d := range resp.Embeddings {
			vec := make([]float32, len(d))
			for j, v := range d {
				vec[j] = float32(v)
			}
			result[i] = vec
		}
		return nil
	}

	fb := func(err error) error {
		logger.Warn().Err(err).Msg("Ollama batch embedder circuit breaker tripped")
		return err
	}

	err := a.embedderCB.Execute(ctx, op, fb)
	if err != nil {
		return nil, err
	}
	return result, nil
}
