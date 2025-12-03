package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/charmbracelet/log"
)

type PerplexityProvider struct {
	apiKey string
	client *http.Client
	logger *log.Logger
}

func NewPerplexityProvider(apiKey string) (*PerplexityProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("perplexity API key is required")
	}

	return &PerplexityProvider{
		apiKey: apiKey,
		client: &http.Client{},
		logger: log.NewWithOptions(os.Stderr, log.Options{
			Prefix: "[perplexity]",
			Level:  log.DebugLevel,
		}),
	}, nil
}

func (p *PerplexityProvider) GetModelName() string {
	return "sonar-pro"
}

func (p *PerplexityProvider) CalculateCost(promptTokens, completionTokens int) float64 {
	// Sonar Pro pricing: $3.00 / 1M input tokens, $15.00 / 1M output tokens
	promptCost := float64(promptTokens) * 3.0 / 1_000_000
	completionCost := float64(completionTokens) * 15.0 / 1_000_000
	return promptCost + completionCost
}

func (p *PerplexityProvider) AnalyzeImage(ctx context.Context, req ImageAnalysisRequest) (*AnalysisResponse, error) {
	langInstruction := LanguageInstructions[req.Language]
	if langInstruction == "" {
		langInstruction = LanguageInstructions["en"]
	}

	var systemPrompt string
	var userTextContent string

	if req.TotalWeight == nil {
		systemPrompt = fmt.Sprintf(`You are a nutrition analysis assistant. %s

Analyze the food image and provide detailed nutritional information.
If the image is NOT food-related or is inappropriate, respond with: {"violation": true, "reason": "not food-related"}
Otherwise, estimate the portion size/weight and provide accurate nutritional values per 100g AND for the estimated total weight.

Response format (JSON):
{
  "productName": "Name of the food item",
  "confidence": 0.0-1.0,
  "explanation": "Brief explanation of the food item and portion size estimation",
  "estimatedWeight": estimated weight in grams or milliliters,
  "weightUnit": "grams" or "milliliters",
  "basicCalories": calories per 100g,
  "basicProtein": protein per 100g,
  "basicFat": fat per 100g,
  "basicCarbs": carbs per 100g,
  "calories": total calories for estimated weight,
  "protein": total protein for estimated weight,
  "fat": total fat for estimated weight,
  "carbs": total carbs for estimated weight
}`, langInstruction)
		userTextContent = req.UserPrompt
	} else {
		systemPrompt = fmt.Sprintf(`You are a nutrition analysis assistant. %s

Analyze the food image and provide detailed nutritional information.
If the image is NOT food-related or is inappropriate, respond with: {"violation": true, "reason": "not food-related"}
Otherwise, provide accurate nutritional values per 100g AND for the total weight specified by the user.

Response format (JSON):
{
  "productName": "Name of the food item",
  "confidence": 0.0-1.0,
  "explanation": "Brief explanation of the food item",
  "basicCalories": calories per 100g,
  "basicProtein": protein per 100g,
  "basicFat": fat per 100g,
  "basicCarbs": carbs per 100g,
  "calories": total calories for specified weight,
  "protein": total protein for specified weight,
  "fat": total fat for specified weight,
  "carbs": total carbs for specified weight
}`, langInstruction)
		userTextContent = fmt.Sprintf("%s\n\nTotal weight: %.1fg", req.UserPrompt, *req.TotalWeight)
	}

	// Build multimodal user message content with base64 image
	imageDataURL := fmt.Sprintf("data:image/jpeg;base64,%s", req.ImageBase64)
	userContent := []map[string]interface{}{
		{
			"type": "text",
			"text": userTextContent,
		},
		{
			"type": "image_url",
			"image_url": map[string]string{
				"url": imageDataURL,
			},
		},
	}

	payload := map[string]interface{}{
		"model": "sonar-pro",
		"messages": []map[string]interface{}{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userContent},
		},
		"max_tokens":  500,
		"temperature": 0.3,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		"https://api.perplexity.ai/chat/completions", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, body)
	}

	var apiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("no response from perplexity")
	}

	content := apiResp.Choices[0].Message.Content

	// Check for violations first
	if p.detectViolation(content) {
		return &AnalysisResponse{
			IsViolation:      true,
			ViolationReason:  "Content is not food-related or inappropriate",
			PromptTokens:     apiResp.Usage.PromptTokens,
			CompletionTokens: apiResp.Usage.CompletionTokens,
		}, nil
	}

	var result FoodAnalysisResult
	err = ParseJSONResponse(content, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &AnalysisResponse{
		ProductName:      result.ProductName,
		Confidence:       result.Confidence,
		Explanation:      result.Explanation,
		BasicCalories:    result.BasicCalories,
		BasicProtein:     result.BasicProtein,
		BasicFat:         result.BasicFat,
		BasicCarbs:       result.BasicCarbs,
		Calories:         result.Calories,
		Protein:          result.Protein,
		Fat:              result.Fat,
		Carbs:            result.Carbs,
		EstimatedWeight:  result.EstimatedWeight,
		WeightUnit:       result.WeightUnit,
		PromptTokens:     apiResp.Usage.PromptTokens,
		CompletionTokens: apiResp.Usage.CompletionTokens,
		IsViolation:      false,
	}, nil
}

// detectViolation checks if response indicates a violation
func (p *PerplexityProvider) detectViolation(content string) bool {
	lowerContent := strings.ToLower(content)
	return strings.Contains(lowerContent, `"violation": true`) ||
		strings.Contains(lowerContent, "not food-related") ||
		strings.Contains(lowerContent, "inappropriate")
}

func (p *PerplexityProvider) AnalyzeText(ctx context.Context, req TextAnalysisRequest) (*AnalysisResponse, error) {
	langInstruction := LanguageInstructions[req.Language]
	if langInstruction == "" {
		langInstruction = LanguageInstructions["en"]
	}

	systemPrompt := fmt.Sprintf(`You are a nutrition analysis assistant. %s

Analyze the food based on its name and description. Provide accurate nutritional values per 100g AND for the total weight.

Response format (JSON):
{
  "productName": "Name of the food item",
  "confidence": 0.0-1.0,
  "explanation": "Brief explanation",
  "basicCalories": calories per 100g,
  "basicProtein": protein per 100g,
  "basicFat": fat per 100g,
  "basicCarbs": carbs per 100g,
  "calories": total calories,
  "protein": total protein,
  "fat": total fat,
  "carbs": total carbs
}`, langInstruction)

	userMessage := fmt.Sprintf("Food: %s\nDescription: %s\nWeight: %.1fg", req.FoodName, req.FoodDescription, req.TotalWeight)

	payload := map[string]interface{}{
		"model": "sonar-pro",
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userMessage},
		},
		"max_tokens":  500,
		"temperature": 0.3,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		"https://api.perplexity.ai/chat/completions", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, body)
	}

	var apiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("no response from perplexity")
	}

	content := apiResp.Choices[0].Message.Content

	var result FoodAnalysisResult
	err = ParseJSONResponse(content, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &AnalysisResponse{
		ProductName:      result.ProductName,
		Confidence:       result.Confidence,
		Explanation:      result.Explanation,
		BasicCalories:    result.BasicCalories,
		BasicProtein:     result.BasicProtein,
		BasicFat:         result.BasicFat,
		BasicCarbs:       result.BasicCarbs,
		Calories:         result.Calories,
		Protein:          result.Protein,
		Fat:              result.Fat,
		Carbs:            result.Carbs,
		PromptTokens:     apiResp.Usage.PromptTokens,
		CompletionTokens: apiResp.Usage.CompletionTokens,
		IsViolation:      false,
	}, nil
}
