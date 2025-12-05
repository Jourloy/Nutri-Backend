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

// getLanguageDescription returns language instruction for JSON schema descriptions
func getLanguageDescription(language string) string {
	switch language {
	case "ru":
		return "на русском языке"
	case "es":
		return "en español"
	case "de":
		return "auf Deutsch"
	case "fr":
		return "en français"
	default:
		return "in English"
	}
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
If the image is NOT food-related or is inappropriate, set violation to true.
Otherwise, estimate the portion size/weight and provide accurate nutritional values per 100g AND for the estimated total weight.`, langInstruction)
		userTextContent = req.UserPrompt
	} else {
		systemPrompt = fmt.Sprintf(`You are a nutrition analysis assistant. %s

Analyze the food image and provide detailed nutritional information.
If the image is NOT food-related or is inappropriate, set violation to true.
Otherwise, provide accurate nutritional values per 100g AND for the total weight specified by the user.`, langInstruction)
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

	// Define JSON schema for structured output
	langDesc := getLanguageDescription(req.Language)
	jsonSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"violation": map[string]interface{}{
				"type": "boolean",
			},
			"reason": map[string]interface{}{
				"type":        "string",
				"description": fmt.Sprintf("Reason for violation (%s)", langDesc),
			},
			"productName": map[string]interface{}{
				"type":        "string",
				"description": fmt.Sprintf("Name of the food item (%s)", langDesc),
			},
			"confidence": map[string]interface{}{
				"type": "number",
			},
			"explanation": map[string]interface{}{
				"type":        "string",
				"description": fmt.Sprintf("Brief explanation (%s)", langDesc),
			},
			"estimatedWeight": map[string]interface{}{
				"type": "number",
			},
			"weightUnit": map[string]interface{}{
				"type":        "string",
				"description": fmt.Sprintf("Weight unit (%s), e.g. 'граммы' or 'миллилитры'", langDesc),
			},
			"basicCalories": map[string]interface{}{
				"type": "number",
			},
			"basicProtein": map[string]interface{}{
				"type": "number",
			},
			"basicFat": map[string]interface{}{
				"type": "number",
			},
			"basicCarbs": map[string]interface{}{
				"type": "number",
			},
			"calories": map[string]interface{}{
				"type": "number",
			},
			"protein": map[string]interface{}{
				"type": "number",
			},
			"fat": map[string]interface{}{
				"type": "number",
			},
			"carbs": map[string]interface{}{
				"type": "number",
			},
		},
		"required":             []string{},
		"additionalProperties": false,
	}

	payload := map[string]interface{}{
		"model": "sonar-pro",
		"messages": []map[string]interface{}{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userContent},
		},
		"max_tokens":  500,
		"temperature": 0.3,
		"response_format": map[string]interface{}{
			"type": "json_schema",
			"json_schema": map[string]interface{}{
				"name":   "food_analysis",
				"schema": jsonSchema,
				"strict": true,
			},
		},
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
	p.logger.Debug("raw AI response", "content", content)

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
		p.logger.Error("failed to parse AI response", "error", err, "rawContent", content)
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

Analyze the food based on its name and description. Provide accurate nutritional values per 100g AND for the total weight.`, langInstruction)

	userMessage := fmt.Sprintf("Food: %s\nDescription: %s\nWeight: %.1fg", req.FoodName, req.FoodDescription, req.TotalWeight)

	// Define JSON schema for structured output
	langDesc := getLanguageDescription(req.Language)
	jsonSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"productName": map[string]interface{}{
				"type":        "string",
				"description": fmt.Sprintf("Name of the food item (%s)", langDesc),
			},
			"confidence": map[string]interface{}{
				"type": "number",
			},
			"explanation": map[string]interface{}{
				"type":        "string",
				"description": fmt.Sprintf("Brief explanation (%s)", langDesc),
			},
			"basicCalories": map[string]interface{}{
				"type": "number",
			},
			"basicProtein": map[string]interface{}{
				"type": "number",
			},
			"basicFat": map[string]interface{}{
				"type": "number",
			},
			"basicCarbs": map[string]interface{}{
				"type": "number",
			},
			"calories": map[string]interface{}{
				"type": "number",
			},
			"protein": map[string]interface{}{
				"type": "number",
			},
			"fat": map[string]interface{}{
				"type": "number",
			},
			"carbs": map[string]interface{}{
				"type": "number",
			},
		},
		"required":             []string{},
		"additionalProperties": false,
	}

	payload := map[string]interface{}{
		"model": "sonar-pro",
		"messages": []map[string]interface{}{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userMessage},
		},
		"max_tokens":  500,
		"temperature": 0.3,
		"response_format": map[string]interface{}{
			"type": "json_schema",
			"json_schema": map[string]interface{}{
				"name":   "food_analysis",
				"schema": jsonSchema,
				"strict": true,
			},
		},
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
	p.logger.Debug("raw AI response", "content", content)

	var result FoodAnalysisResult
	err = ParseJSONResponse(content, &result)
	if err != nil {
		p.logger.Error("failed to parse AI response", "error", err, "rawContent", content)
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
