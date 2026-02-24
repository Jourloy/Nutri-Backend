package ai

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/sashabaranov/go-openai"
)

type OpenAIProvider struct {
	client *openai.Client
	logger *log.Logger
}

func NewOpenAIProvider(apiKey string) (*OpenAIProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("openai API key is required")
	}

	return &OpenAIProvider{
		client: openai.NewClient(apiKey),
		logger: log.NewWithOptions(os.Stderr, log.Options{
			Prefix: "[openai]",
			Level:  log.DebugLevel,
		}),
	}, nil
}

func (p *OpenAIProvider) GetModelName() string {
	return "gpt-4o"
}

func (p *OpenAIProvider) CalculateCost(promptTokens, completionTokens int) float64 {
	// GPT-4o pricing: $5.00 / 1M input tokens, $15.00 / 1M output tokens
	promptCost := float64(promptTokens) * 5.0 / 1_000_000
	completionCost := float64(completionTokens) * 15.0 / 1_000_000
	return promptCost + completionCost
}

func (p *OpenAIProvider) AnalyzeImage(ctx context.Context, req ImageAnalysisRequest) (*AnalysisResponse, error) {
	langInstruction := LanguageInstructions[req.Language]
	if langInstruction == "" {
		langInstruction = LanguageInstructions["en"]
	}

	var systemPrompt string
	var userMessage string

	if req.TotalWeight == nil {
		// Weight not provided - ask AI to estimate
		systemPrompt = fmt.Sprintf(`You are a nutrition analysis assistant. %s

Analyze the food image and provide detailed nutritional information.
If the image is NOT food-related or is inappropriate, respond with: {"violation": true, "reason": "not food-related"}

IMPORTANT: The user may provide a product name and/or quantity in their message (e.g., "3 pancakes with condensed milk").
- If the user specified a name/quantity AND the image matches this description, use the user's name and quantity exactly
- If the image clearly shows a DIFFERENT product than described (e.g., user says "pancakes" but image shows soup), identify the actual product from the image
- When in doubt, trust the image over the user's description

Estimate the portion size/weight and provide accurate nutritional values per 100g AND for the estimated total weight.

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
  "basicFiber": fiber per 100g in grams (or null if unknown),
  "basicCholesterol": cholesterol per 100g in mg (or null if unknown),
  "calories": total calories for estimated weight,
  "protein": total protein for estimated weight,
  "fat": total fat for estimated weight,
  "carbs": total carbs for estimated weight,
  "fiber": total fiber for estimated weight in grams (or null if unknown),
  "cholesterol": total cholesterol for estimated weight in mg (or null if unknown)
}`, langInstruction)
		userMessage = req.UserPrompt
	} else {
		// Weight provided by user
		systemPrompt = fmt.Sprintf(`You are a nutrition analysis assistant. %s

Analyze the food image and provide detailed nutritional information.
If the image is NOT food-related or is inappropriate, respond with: {"violation": true, "reason": "not food-related"}

IMPORTANT: The user may provide a product name and/or quantity in their message (e.g., "3 pancakes with condensed milk").
- If the user specified a name/quantity AND the image matches this description, use the user's name and quantity exactly
- If the image clearly shows a DIFFERENT product than described (e.g., user says "pancakes" but image shows soup), identify the actual product from the image
- When in doubt, trust the image over the user's description

Provide accurate nutritional values per 100g AND for the total weight specified by the user.

Response format (JSON):
{
  "productName": "Name of the food item",
  "confidence": 0.0-1.0,
  "explanation": "Brief explanation of the food item",
  "basicCalories": calories per 100g,
  "basicProtein": protein per 100g,
  "basicFat": fat per 100g,
  "basicCarbs": carbs per 100g,
  "basicFiber": fiber per 100g in grams (or null if unknown),
  "basicCholesterol": cholesterol per 100g in mg (or null if unknown),
  "calories": total calories for specified weight,
  "protein": total protein for specified weight,
  "fat": total fat for specified weight,
  "carbs": total carbs for specified weight,
  "fiber": total fiber for specified weight in grams (or null if unknown),
  "cholesterol": total cholesterol for specified weight in mg (or null if unknown)
}`, langInstruction)
		userMessage = fmt.Sprintf("%s\n\nTotal weight: %.1fg", req.UserPrompt, *req.TotalWeight)
	}

	resp, err := p.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: openai.GPT4o,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: systemPrompt,
				},
				{
					Role: openai.ChatMessageRoleUser,
					MultiContent: []openai.ChatMessagePart{
						{
							Type: openai.ChatMessagePartTypeText,
							Text: userMessage,
						},
						{
							Type: openai.ChatMessagePartTypeImageURL,
							ImageURL: &openai.ChatMessageImageURL{
								URL:    req.ImageURL,
								Detail: openai.ImageURLDetailAuto,
							},
						},
					},
				},
			},
			MaxTokens:   500,
			Temperature: 0.3,
		},
	)

	if err != nil {
		return nil, err
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from openai")
	}

	content := resp.Choices[0].Message.Content

	// Check for violations first
	if p.detectViolation(content) {
		return &AnalysisResponse{
			IsViolation:      true,
			ViolationReason:  "Content is not food-related or inappropriate",
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
		}, nil
	}

	// Parse response
	var result FoodAnalysisResult
	err = ParseJSONResponse(content, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse openai response: %w", err)
	}

	return &AnalysisResponse{
		ProductName:      result.ProductName,
		Confidence:       result.Confidence,
		Explanation:      result.Explanation,
		BasicCalories:    result.BasicCalories,
		BasicProtein:     result.BasicProtein,
		BasicFat:         result.BasicFat,
		BasicCarbs:       result.BasicCarbs,
		BasicFiber:       result.BasicFiber,
		BasicCholesterol: result.BasicCholesterol,
		Calories:         result.Calories,
		Protein:          result.Protein,
		Fat:              result.Fat,
		Carbs:            result.Carbs,
		Fiber:            result.Fiber,
		Cholesterol:      result.Cholesterol,
		EstimatedWeight:  result.EstimatedWeight,
		WeightUnit:       result.WeightUnit,
		PromptTokens:     resp.Usage.PromptTokens,
		CompletionTokens: resp.Usage.CompletionTokens,
		IsViolation:      false,
	}, nil
}

func (p *OpenAIProvider) AnalyzeText(ctx context.Context, req TextAnalysisRequest) (*AnalysisResponse, error) {
	langInstruction := LanguageInstructions[req.Language]
	if langInstruction == "" {
		langInstruction = LanguageInstructions["en"]
	}

	systemPrompt := fmt.Sprintf(`You are a nutrition analysis assistant. %s

Analyze the food based on its name and description, and provide detailed nutritional information.
Provide accurate nutritional values per 100g AND for the total weight specified by the user.

Response format (JSON):
{
  "productName": "Name of the food item",
  "confidence": 0.0-1.0,
  "explanation": "Brief explanation about the nutritional analysis",
  "basicCalories": calories per 100g,
  "basicProtein": protein per 100g,
  "basicFat": fat per 100g,
  "basicCarbs": carbs per 100g,
  "basicFiber": fiber per 100g in grams (or null if unknown),
  "basicCholesterol": cholesterol per 100g in mg (or null if unknown),
  "calories": total calories for specified weight,
  "protein": total protein for specified weight,
  "fat": total fat for specified weight,
  "carbs": total carbs for specified weight,
  "fiber": total fiber for specified weight in grams (or null if unknown),
  "cholesterol": total cholesterol for specified weight in mg (or null if unknown)
}`, langInstruction)

	userMessage := fmt.Sprintf("Food name: %s\nDescription: %s\nTotal weight: %.1fg\n\nProvide nutritional information for this food.", req.FoodName, req.FoodDescription, req.TotalWeight)

	resp, err := p.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: openai.GPT4o,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: systemPrompt,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: userMessage,
				},
			},
			MaxTokens:   500,
			Temperature: 0.3,
		},
	)

	if err != nil {
		return nil, err
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from openai")
	}

	content := resp.Choices[0].Message.Content

	// Parse response
	var result FoodAnalysisResult
	err = ParseJSONResponse(content, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse openai response: %w", err)
	}

	return &AnalysisResponse{
		ProductName:      result.ProductName,
		Confidence:       result.Confidence,
		Explanation:      result.Explanation,
		BasicCalories:    result.BasicCalories,
		BasicProtein:     result.BasicProtein,
		BasicFat:         result.BasicFat,
		BasicCarbs:       result.BasicCarbs,
		BasicFiber:       result.BasicFiber,
		BasicCholesterol: result.BasicCholesterol,
		Calories:         result.Calories,
		Protein:          result.Protein,
		Fat:              result.Fat,
		Carbs:            result.Carbs,
		Fiber:            result.Fiber,
		Cholesterol:      result.Cholesterol,
		PromptTokens:     resp.Usage.PromptTokens,
		CompletionTokens: resp.Usage.CompletionTokens,
		IsViolation:      false,
	}, nil
}

// detectViolation checks if response indicates a violation
func (p *OpenAIProvider) detectViolation(content string) bool {
	lowerContent := strings.ToLower(content)
	return strings.Contains(lowerContent, `"violation": true`) ||
		strings.Contains(lowerContent, "not food-related") ||
		strings.Contains(lowerContent, "inappropriate")
}

func (p *OpenAIProvider) ImproveText(ctx context.Context, html string) (string, error) {
	resp, err := p.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: openai.GPT4o,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: improveTextSystemPrompt,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: html,
				},
			},
			MaxTokens:   2000,
			Temperature: 0.2,
		},
	)
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from openai")
	}

	content := StripMarkdownCodeFences(resp.Choices[0].Message.Content)
	if content == "" {
		return "", fmt.Errorf("empty response from openai")
	}
	return content, nil
}

func (p *OpenAIProvider) GenerateArticle(ctx context.Context, req GenerateArticleRequest) (*GeneratedArticle, error) {
	userMessage := fmt.Sprintf("Topic: %s\nDescription: %s", req.Topic, req.Description)

	resp, err := p.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: openai.GPT4o,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: generateArticleSystemPrompt,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: userMessage,
				},
			},
			MaxTokens:   8000,
			Temperature: 0.4,
		},
	)
	if err != nil {
		return nil, err
	}
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from openai")
	}

	content := StripMarkdownCodeFences(resp.Choices[0].Message.Content)
	if content == "" {
		return nil, fmt.Errorf("empty response from openai")
	}
	var article GeneratedArticle
	if err := ParseJSONResponse(content, &article); err != nil {
		return nil, fmt.Errorf("failed to parse openai response: %w", err)
	}

	if article.TitleRu == "" || article.TitleEn == "" || article.ContentRu == "" || article.ContentEn == "" {
		return nil, fmt.Errorf("incomplete article generated")
	}

	return &article, nil
}

func (p *OpenAIProvider) GenerateRecipeDraft(ctx context.Context, req GenerateRecipeDraftRequest) (*GeneratedRecipeDraft, error) {
	imageURL := strings.TrimSpace(req.ImageURL)
	if imageBase64 := strings.TrimSpace(req.ImageBase64); imageBase64 != "" {
		// Prefer inline base64 to avoid provider-side fetch issues with private/presigned URLs.
		imageURL = "data:image/jpeg;base64," + imageBase64
	}
	if imageURL == "" {
		return nil, fmt.Errorf("image is required")
	}

	userMessage := fmt.Sprintf("Title (RU): %s\n\nIngredients (RU):\n%s\n\nSteps (RU):\n%s",
		strings.TrimSpace(req.TitleRu),
		strings.TrimSpace(req.IngredientsRu),
		strings.TrimSpace(req.StepsRu),
	)

	resp, err := p.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: openai.GPT4o,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: generateRecipeDraftSystemPrompt,
				},
				{
					Role: openai.ChatMessageRoleUser,
					MultiContent: []openai.ChatMessagePart{
						{
							Type: openai.ChatMessagePartTypeText,
							Text: userMessage,
						},
						{
							Type: openai.ChatMessagePartTypeImageURL,
							ImageURL: &openai.ChatMessageImageURL{
								URL:    imageURL,
								Detail: openai.ImageURLDetailAuto,
							},
						},
					},
				},
			},
			MaxTokens:   3500,
			Temperature: 0.2,
		},
	)
	if err != nil {
		return nil, err
	}
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from openai")
	}

	content := StripMarkdownCodeFences(resp.Choices[0].Message.Content)
	if content == "" {
		return nil, fmt.Errorf("empty response from openai")
	}

	var draft GeneratedRecipeDraft
	if err := ParseJSONResponse(content, &draft); err != nil {
		return nil, fmt.Errorf("failed to parse openai response: %w", err)
	}

	draftNormalized := NormalizeGeneratedRecipeDraft(&draft)
	if draftNormalized == nil {
		return nil, fmt.Errorf("empty recipe draft")
	}
	if len(draftNormalized.Ingredients) == 0 || len(draftNormalized.Steps) == 0 {
		return nil, fmt.Errorf("incomplete recipe draft generated")
	}

	return draftNormalized, nil
}
