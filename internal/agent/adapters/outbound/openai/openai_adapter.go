package openai

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/morphy76/tacito-square/internal/agent/adapters/outbound/resiliency"
	"github.com/morphy76/tacito-square/internal/agent/domain/model"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/shared"
	"github.com/rs/zerolog"
	"github.com/sethvargo/go-retry"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
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
}

type Adapter struct {
	cfg    Config
	client openai.Client
	cb     *resiliency.CircuitBreaker
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
	}
	if cfg.Endpoint != "" {
		opts = append(opts, option.WithBaseURL(cfg.Endpoint))
	}

	client := openai.NewClient(opts...)
	cb := resiliency.NewCircuitBreaker(cfg.FailureThreshold, cfg.RecoveryTimeout)

	return &Adapter{
		cfg:    cfg,
		client: client,
		cb:     cb,
	}
}

func (a *Adapter) Generate(ctx context.Context, req model.BrainRequest) (*model.BrainResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("model", a.cfg.Model).
		Str("endpoint", a.cfg.Endpoint).
		Msg("sending chat completion request to OpenAI")

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
		messages = append(messages, openai.UserMessage(req.Prompt))

		modelName := a.cfg.Model
		if modelName == "" {
			modelName = "gpt-4o"
		}

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

		logger.Info().Msg("initiating chat completion wire call (with backoff retry)")
		var chatComp *openai.ChatCompletion
		start := time.Now()
		err := retry.Do(runCtx, b, func(ctx context.Context) error {
			var err error
			chatComp, err = a.client.Chat.Completions.New(ctx, params)
			if err != nil {
				logger.Warn().Err(err).Msg("chat completion call failed, retrying...")
				return retry.RetryableError(err)
			}
			return nil
		})
		duration := time.Since(start)

		if err != nil {
			logger.Error().Err(err).Dur("duration_ms", duration).Msg("all chat completion retries exhausted")
			return err
		}

		if len(chatComp.Choices) == 0 {
			return errors.New("empty choices from openai response")
		}

		logger.Info().
			Dur("duration_ms", duration).
			Int("prompt_tokens", int(chatComp.Usage.PromptTokens)).
			Int("completion_tokens", int(chatComp.Usage.CompletionTokens)).
			Msg("chat completion wire call completed successfully")

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
