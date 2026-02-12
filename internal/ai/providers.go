package ai

import (
	"context"
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/jourloy/nutri-backend/internal/lib"
)

// AIProvider interface defines the methods that all AI providers must implement
// This is used for documentation purposes; actual implementations use duck typing
type AIProvider interface {
	AnalyzeImage(ctx context.Context, req ImageAnalysisRequest) (*AnalysisResponse, error)
	AnalyzeText(ctx context.Context, req TextAnalysisRequest) (*AnalysisResponse, error)
	ImproveText(ctx context.Context, html string) (string, error)
	GenerateArticle(ctx context.Context, req GenerateArticleRequest) (*GeneratedArticle, error)
	GetModelName() string
	CalculateCost(promptTokens, completionTokens int) float64
}

// GetProvider returns the configured AI provider with fallback logic
func GetProvider(logger *log.Logger) (AIProvider, error) {
	providerName := lib.Config.AIProvider

	switch providerName {
	case "perplexity":
		provider, err := NewPerplexityProvider(lib.Config.PerplexityAPIKey)
		if err != nil {
			logger.Error("failed to initialize Perplexity", "error", err)
			// Try fallback to OpenAI if available
			if lib.Config.OpenAIAPIKey != "" {
				logger.Warn("falling back to OpenAI provider")
				return NewOpenAIProvider(lib.Config.OpenAIAPIKey)
			}
			return nil, fmt.Errorf("failed to initialize Perplexity: %w", err)
		}
		logger.Info("using Perplexity provider", "model", provider.GetModelName())
		return provider, nil

	case "openai":
		provider, err := NewOpenAIProvider(lib.Config.OpenAIAPIKey)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize OpenAI: %w", err)
		}
		logger.Info("using OpenAI provider", "model", provider.GetModelName())
		return provider, nil

	case "auto":
		// Try Perplexity first, then OpenAI
		if lib.Config.PerplexityAPIKey != "" {
			provider, err := NewPerplexityProvider(lib.Config.PerplexityAPIKey)
			if err == nil {
				logger.Info("using Perplexity provider (auto mode)", "model", provider.GetModelName())
				return provider, nil
			}
			logger.Warn("Perplexity initialization failed, trying OpenAI", "error", err)
		}

		if lib.Config.OpenAIAPIKey != "" {
			provider, err := NewOpenAIProvider(lib.Config.OpenAIAPIKey)
			if err == nil {
				logger.Info("using OpenAI provider (auto mode)", "model", provider.GetModelName())
				return provider, nil
			}
			logger.Error("OpenAI initialization failed", "error", err)
		}

		return nil, fmt.Errorf("no AI providers available (tried: Perplexity, OpenAI)")

	default:
		return nil, fmt.Errorf("unknown AI provider: %s (supported: perplexity, openai, auto)", providerName)
	}
}

// GetFallbackProvider returns an alternate provider (if configured) that can be used as a one-shot fallback.
// It never returns the same provider type as primary.
func GetFallbackProvider(logger *log.Logger, primary AIProvider) (AIProvider, error) {
	switch primary.(type) {
	case *PerplexityProvider:
		if lib.Config.OpenAIAPIKey == "" {
			return nil, nil
		}
		p, err := NewOpenAIProvider(lib.Config.OpenAIAPIKey)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize OpenAI fallback: %w", err)
		}
		logger.Info("AI fallback provider initialized", "provider", "openai", "model", p.GetModelName())
		return p, nil

	case *OpenAIProvider:
		if lib.Config.PerplexityAPIKey == "" {
			return nil, nil
		}
		p, err := NewPerplexityProvider(lib.Config.PerplexityAPIKey)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Perplexity fallback: %w", err)
		}
		logger.Info("AI fallback provider initialized", "provider", "perplexity", "model", p.GetModelName())
		return p, nil

	default:
		return nil, nil
	}
}
