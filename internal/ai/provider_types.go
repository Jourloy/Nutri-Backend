package ai

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ImageAnalysisRequest represents a request for image-based food analysis
type ImageAnalysisRequest struct {
	ImageURL    string
	ImageBase64 string // base64 encoded image data (for providers that don't accept URLs)
	UserPrompt  string
	TotalWeight *float64 // nil if AI should estimate
	Language    string
}

// TextAnalysisRequest represents a request for text-based food analysis
type TextAnalysisRequest struct {
	FoodName        string
	FoodDescription string
	TotalWeight     float64
	Language        string
}

// AnalysisResponse represents a provider-agnostic response from AI analysis
type AnalysisResponse struct {
	ProductName      string
	Confidence       float64
	Explanation      string
	BasicCalories    float64 // per 100g
	BasicProtein     float64 // per 100g
	BasicFat         float64 // per 100g
	BasicCarbs       float64 // per 100g
	Calories         float64 // for total weight
	Protein          float64 // for total weight
	Fat              float64 // for total weight
	Carbs            float64 // for total weight
	EstimatedWeight  *float64
	WeightUnit       *string
	PromptTokens     int
	CompletionTokens int
	IsViolation      bool
	ViolationReason  string
}

// LanguageInstructions provides language-specific instructions for AI prompts
var LanguageInstructions = map[string]string{
	"ru": "Отвечай на русском языке. Все поля JSON должны содержать русский текст.",
	"en": "Respond in English. All JSON fields should contain English text.",
	"es": "Responde en español. Todos los campos JSON deben contener texto en español.",
	"de": "Antworte auf Deutsch. Alle JSON-Felder sollten deutschen Text enthalten.",
	"fr": "Répondez en français. Tous les champs JSON doivent contenir du texte en français.",
}

// ParseJSONResponse attempts to parse JSON from content, handling markdown code blocks
func ParseJSONResponse(content string, result interface{}) error {
	// Try direct parsing first
	err := json.Unmarshal([]byte(content), result)
	if err == nil {
		return nil
	}

	// Try to extract JSON from markdown code block
	if strings.Contains(content, "```json") {
		start := strings.Index(content, "```json") + 7
		end := strings.LastIndex(content, "```")
		if end > start {
			jsonStr := strings.TrimSpace(content[start:end])
			err = json.Unmarshal([]byte(jsonStr), result)
			if err == nil {
				return nil
			}
		}
	}

	// Try to extract from ```
	if strings.Contains(content, "```") {
		start := strings.Index(content, "```") + 3
		end := strings.LastIndex(content, "```")
		if end > start {
			jsonStr := strings.TrimSpace(content[start:end])
			err = json.Unmarshal([]byte(jsonStr), result)
			if err == nil {
				return nil
			}
		}
	}

	return fmt.Errorf("failed to parse JSON from response: %w", err)
}
