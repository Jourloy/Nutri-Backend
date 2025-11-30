package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/nfnt/resize"
	"github.com/sashabaranov/go-openai"

	"github.com/jourloy/nutri-backend/internal/lib"
)

type Service interface {
	AnalyzeFoodImage(ctx context.Context, userId string, imageData []byte, totalWeight *float64, userPrompt string, language string) (*FoodAnalysisResult, error)
	AnalyzeFoodByText(ctx context.Context, userId string, foodName string, foodDescription string, totalWeight float64, language string) (*FoodAnalysisResult, error)
	CheckUserLimit(ctx context.Context, userId, requestType string) (*LimitCheckResult, error)
	GetUserAnalysisHistory(ctx context.Context, userId string, limit int) ([]AnalysisLog, error)
}

type service struct {
	repo         Repository
	openaiClient *openai.Client
	minioClient  *minio.Client
	logger       *log.Logger
}

func NewService(repo Repository) (Service, error) {
	// Initialize OpenAI client
	openaiClient := openai.NewClient(lib.Config.OpenAIAPIKey)

	// Parse Minio endpoint to extract hostname and determine SSL
	endpoint := lib.Config.MinioEndpoint
	useSSL := lib.Config.MinioUseSSL

	// If endpoint contains protocol, parse it
	if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
		parsedURL, err := url.Parse(endpoint)
		if err != nil {
			return nil, fmt.Errorf("failed to parse minio endpoint: %w", err)
		}
		endpoint = parsedURL.Host
		useSSL = parsedURL.Scheme == "https"
	}

	// Initialize Minio client
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(lib.Config.MinioAccessKey, lib.Config.MinioSecretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize minio client: %w", err)
	}

	// Initialize logger first
	logger := log.NewWithOptions(os.Stderr, log.Options{
		Prefix: "[ai-svc]",
		Level:  log.DebugLevel,
	})

	// Ensure bucket exists - try to create it, ignore if already exists
	bucketName := lib.Config.MinioBucketName
	err = minioClient.MakeBucket(context.Background(), bucketName, minio.MakeBucketOptions{})
	if err != nil {
		// Check if bucket already exists
		exists, errBucketExists := minioClient.BucketExists(context.Background(), bucketName)
		if errBucketExists != nil {
			logger.Warn("failed to check bucket existence", "bucket", bucketName, "error", errBucketExists)
		}
		if !exists {
			logger.Warn("bucket does not exist and could not be created - please create it manually", "bucket", bucketName, "error", err)
			logger.Info("AI service will continue, but image uploads will fail until bucket is created")
		} else {
			logger.Debug("bucket already exists", "bucket", bucketName)
		}
	} else {
		logger.Info("bucket created successfully", "bucket", bucketName)
	}

	return &service{
		repo:         repo,
		openaiClient: openaiClient,
		minioClient:  minioClient,
		logger:       logger,
	}, nil
}

// AnalyzeFoodImage performs AI analysis of food image with full workflow
func (s *service) AnalyzeFoodImage(ctx context.Context, userId string, imageData []byte, totalWeight *float64, userPrompt string, language string) (*FoodAnalysisResult, error) {
	startTime := time.Now()
	requestType := "food_analysis"

	// Default to English if language not specified
	if language == "" {
		language = "en"
	}

	// 1. Check if user is banned
	banUntil, err := s.repo.GetUserBanStatus(ctx, userId)
	if err != nil {
		s.logger.Error("failed to check ban status", "userId", userId, "error", err)
	}
	if banUntil != nil && banUntil.After(time.Now()) {
		return nil, fmt.Errorf("user is banned until %s", banUntil.Format("2006-01-02 15:04:05"))
	}

	// 2. Check usage limits
	limitResult, err := s.CheckUserLimit(ctx, userId, requestType)
	if err != nil {
		return nil, fmt.Errorf("failed to check user limit: %w", err)
	}
	if !limitResult.Allowed {
		return nil, fmt.Errorf("daily limit exceeded: %s", limitResult.Message)
	}

	// 3. Create initial analysis log
	analysisLog := AnalysisLog{
		UserId:      userId,
		RequestType: requestType,
		UserPrompt:  userPrompt,
		TotalWeight: totalWeight, // May be nil if not provided
		ModelUsed:   "gpt-4o",
		Status:      "pending",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	createdLog, err := s.repo.CreateAnalysisLog(ctx, analysisLog)
	if err != nil {
		return nil, fmt.Errorf("failed to create analysis log: %w", err)
	}

	// 4. Process and upload image to Minio
	imageUrl, err := s.uploadImageToMinio(ctx, userId, imageData)
	if err != nil {
		s.updateLogWithError(ctx, createdLog.Id, "failed to upload image", err)
		return nil, fmt.Errorf("failed to upload image: %w", err)
	}
	createdLog.ImageUrl = imageUrl

	// 5. Call OpenAI Vision API
	response, err := s.callOpenAIVision(ctx, imageUrl, userPrompt, totalWeight, language)
	if err != nil {
		s.updateLogWithError(ctx, createdLog.Id, "openai api error", err)
		return nil, fmt.Errorf("openai api error: %w", err)
	}

	// 6. Check for content moderation flags
	if s.detectViolation(response) {
		s.handleViolation(ctx, userId, createdLog.Id, imageUrl, userPrompt, "off_topic", "Content is not food-related or inappropriate")
		s.updateLogWithStatus(ctx, createdLog.Id, "moderated", "Content moderated")
		return nil, fmt.Errorf("content moderated: image does not appear to be food-related")
	}

	// 7. Parse response
	result, err := s.parseOpenAIResponse(response, totalWeight)
	if err != nil {
		s.updateLogWithError(ctx, createdLog.Id, "failed to parse response", err)
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// 8. Calculate costs
	tokensPrompt := response.Usage.PromptTokens
	tokensCompletion := response.Usage.CompletionTokens
	estimatedCost := s.calculateCost(tokensPrompt, tokensCompletion)

	// 9. Update analysis log with success
	responseDataJSON, _ := json.Marshal(response)
	responseDataStr := string(responseDataJSON)
	parsedResultJSON, _ := json.Marshal(result)
	parsedResultStr := string(parsedResultJSON)

	processingTime := int(time.Since(startTime).Milliseconds())

	err = s.repo.UpdateAnalysisLog(ctx, AnalysisLog{
		Id:               createdLog.Id,
		ImageUrl:         imageUrl,
		ResponseData:     &responseDataStr,
		ParsedResult:     &parsedResultStr,
		TokensPrompt:     &tokensPrompt,
		TokensCompletion: &tokensCompletion,
		EstimatedCostUsd: &estimatedCost,
		Status:           "success",
		ProcessingTimeMs: &processingTime,
		UpdatedAt:        time.Now(),
	})
	if err != nil {
		s.logger.Error("failed to update analysis log", "logId", createdLog.Id, "error", err)
	}

	// 10. Increment user limit
	today := time.Now()
	err = s.repo.IncrementUserLimit(ctx, userId, requestType, today)
	if err != nil {
		s.logger.Error("failed to increment user limit", "userId", userId, "error", err)
	}

	s.logger.Info("food analysis completed", "userId", userId, "logId", createdLog.Id, "cost", estimatedCost, "time_ms", processingTime)

	return result, nil
}

// AnalyzeFoodByText performs AI analysis based on text description without image
func (s *service) AnalyzeFoodByText(ctx context.Context, userId string, foodName string, foodDescription string, totalWeight float64, language string) (*FoodAnalysisResult, error) {
	startTime := time.Now()
	requestType := "food_analysis" // Same request type as image analysis

	// Default to English if language not specified
	if language == "" {
		language = "en"
	}

	// 1. Check if user is banned
	banUntil, err := s.repo.GetUserBanStatus(ctx, userId)
	if err != nil {
		s.logger.Error("failed to check ban status", "userId", userId, "error", err)
	}
	if banUntil != nil && banUntil.After(time.Now()) {
		return nil, fmt.Errorf("user is banned until %s", banUntil.Format("2006-01-02 15:04:05"))
	}

	// 2. Check usage limits
	limitResult, err := s.CheckUserLimit(ctx, userId, requestType)
	if err != nil {
		return nil, fmt.Errorf("failed to check user limit: %w", err)
	}
	if !limitResult.Allowed {
		return nil, fmt.Errorf("weekly limit exceeded: %s", limitResult.Message)
	}

	// 3. Create initial analysis log
	userPrompt := fmt.Sprintf("Food: %s. %s", foodName, foodDescription)
	analysisLog := AnalysisLog{
		UserId:      userId,
		RequestType: requestType,
		UserPrompt:  userPrompt,
		TotalWeight: &totalWeight,
		ModelUsed:   "gpt-4o",
		Status:      "pending",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	createdLog, err := s.repo.CreateAnalysisLog(ctx, analysisLog)
	if err != nil {
		return nil, fmt.Errorf("failed to create analysis log: %w", err)
	}

	// 4. Call OpenAI Text API (without image)
	response, err := s.callOpenAIText(ctx, foodName, foodDescription, totalWeight, language)
	if err != nil {
		s.updateLogWithError(ctx, createdLog.Id, "openai api error", err)
		return nil, fmt.Errorf("openai api error: %w", err)
	}

	// 5. Parse response
	result, err := s.parseOpenAIResponse(response, &totalWeight)
	if err != nil {
		s.updateLogWithError(ctx, createdLog.Id, "failed to parse response", err)
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// 6. Calculate costs
	tokensPrompt := response.Usage.PromptTokens
	tokensCompletion := response.Usage.CompletionTokens
	estimatedCost := s.calculateCost(tokensPrompt, tokensCompletion)

	// 7. Update analysis log with success
	responseDataJSON, _ := json.Marshal(response)
	responseDataStr := string(responseDataJSON)
	parsedResultJSON, _ := json.Marshal(result)
	parsedResultStr := string(parsedResultJSON)

	processingTime := int(time.Since(startTime).Milliseconds())

	err = s.repo.UpdateAnalysisLog(ctx, AnalysisLog{
		Id:               createdLog.Id,
		ResponseData:     &responseDataStr,
		ParsedResult:     &parsedResultStr,
		TokensPrompt:     &tokensPrompt,
		TokensCompletion: &tokensCompletion,
		EstimatedCostUsd: &estimatedCost,
		Status:           "success",
		ProcessingTimeMs: &processingTime,
		UpdatedAt:        time.Now(),
	})
	if err != nil {
		s.logger.Error("failed to update analysis log", "logId", createdLog.Id, "error", err)
	}

	// 8. Increment user limit
	today := time.Now()
	err = s.repo.IncrementUserLimit(ctx, userId, requestType, today)
	if err != nil {
		s.logger.Error("failed to increment user limit", "userId", userId, "error", err)
	}

	s.logger.Info("food text analysis completed", "userId", userId, "logId", createdLog.Id, "cost", estimatedCost, "time_ms", processingTime)

	return result, nil
}

// CheckUserLimit verifies if user can make another request
func (s *service) CheckUserLimit(ctx context.Context, userId, requestType string) (*LimitCheckResult, error) {
	today := time.Now()

	// Get or create limit record
	limit, err := s.repo.GetOrCreateUserLimit(ctx, userId, requestType, today)
	if err != nil {
		return nil, err
	}

	allowed := limit.RequestsCount < limit.MaxRequests

	// Calculate reset date based on subscription tier
	var resetAt string
	if limit.SubscriptionTier != nil && *limit.SubscriptionTier != "free" {
		// Premium users: resets tomorrow (daily limit)
		tomorrow := today.AddDate(0, 0, 1)
		resetAt = tomorrow.Format("2006-01-02")
	} else {
		// Free users: resets next Monday (weekly limit)
		daysUntilMonday := (8 - int(today.Weekday())) % 7
		if daysUntilMonday == 0 {
			daysUntilMonday = 7
		}
		nextMonday := today.AddDate(0, 0, daysUntilMonday)
		resetAt = nextMonday.Format("2006-01-02")
	}

	message := ""
	if !allowed {
		if limit.SubscriptionTier != nil && *limit.SubscriptionTier != "free" {
			message = fmt.Sprintf("Daily limit of %d requests reached. Resets at %s", limit.MaxRequests, resetAt)
		} else {
			message = fmt.Sprintf("Weekly limit of %d requests reached. Resets at %s", limit.MaxRequests, resetAt)
		}
	}

	return &LimitCheckResult{
		Allowed:      allowed,
		CurrentUsage: limit.RequestsCount,
		MaxLimit:     limit.MaxRequests,
		ResetAt:      resetAt,
		Message:      message,
	}, nil
}

// GetUserAnalysisHistory retrieves user's analysis history
func (s *service) GetUserAnalysisHistory(ctx context.Context, userId string, limit int) ([]AnalysisLog, error) {
	return s.repo.GetUserAnalysisLogs(ctx, userId, limit)
}

// uploadImageToMinio uploads image to Minio with compression
func (s *service) uploadImageToMinio(ctx context.Context, userId string, imageData []byte) (string, error) {
	// Decode original image
	img, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return "", fmt.Errorf("failed to decode image: %w", err)
	}

	// Resize to max 800px width for cost savings
	resized := resize.Resize(800, 0, img, resize.Lanczos3)

	// Encode with lower quality (60%) for storage savings
	var buf bytes.Buffer
	err = jpeg.Encode(&buf, resized, &jpeg.Options{Quality: 60})
	if err != nil {
		return "", fmt.Errorf("failed to encode image: %w", err)
	}

	// Generate unique filename
	filename := fmt.Sprintf("%s/%s_%d.jpg", userId, uuid.New().String(), time.Now().Unix())

	// Upload to Minio
	_, err = s.minioClient.PutObject(
		ctx,
		lib.Config.MinioBucketName,
		filename,
		&buf,
		int64(buf.Len()),
		minio.PutObjectOptions{
			ContentType: "image/jpeg",
		},
	)
	if err != nil {
		return "", fmt.Errorf("failed to upload to minio: %w", err)
	}

	// Generate presigned URL (valid for 7 days for moderation review)
	presignedURL, err := s.minioClient.PresignedGetObject(ctx, lib.Config.MinioBucketName, filename, 7*24*time.Hour, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned url: %w", err)
	}

	return presignedURL.String(), nil
}

// callOpenAIVision calls OpenAI Vision API
func (s *service) callOpenAIVision(ctx context.Context, imageUrl, userPrompt string, totalWeight *float64, language string) (*openai.ChatCompletionResponse, error) {
	// Language-specific instructions
	languageInstructions := map[string]string{
		"ru": "Отвечай на русском языке. Все поля JSON должны содержать русский текст.",
		"en": "Respond in English. All JSON fields should contain English text.",
		"es": "Responde en español. Todos los campos JSON deben contener texto en español.",
		"de": "Antworte auf Deutsch. Alle JSON-Felder sollten deutschen Text enthalten.",
		"fr": "Répondez en français. Tous les champs JSON doivent contenir du texte en français.",
	}

	langInstruction := languageInstructions[language]
	if langInstruction == "" {
		langInstruction = languageInstructions["en"]
	}

	var systemPrompt string
	var userMessage string

	if totalWeight == nil {
		// Weight not provided - ask AI to estimate
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
		userMessage = userPrompt
	} else {
		// Weight provided by user
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
		userMessage = fmt.Sprintf("%s\n\nTotal weight: %.1fg", userPrompt, *totalWeight)
	}

	resp, err := s.openaiClient.CreateChatCompletion(
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
								URL:    imageUrl,
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

	return &resp, nil
}

// callOpenAIText calls OpenAI API for text-based food analysis
func (s *service) callOpenAIText(ctx context.Context, foodName, foodDescription string, totalWeight float64, language string) (*openai.ChatCompletionResponse, error) {
	// Language-specific instructions
	languageInstructions := map[string]string{
		"ru": "Отвечай на русском языке. Все поля JSON должны содержать русский текст.",
		"en": "Respond in English. All JSON fields should contain English text.",
		"es": "Responde en español. Todos los campos JSON deben contener texto en español.",
		"de": "Antworte auf Deutsch. Alle JSON-Felder sollten deutschen Text enthalten.",
		"fr": "Répondez en français. Tous les champs JSON doivent contenir du texte en français.",
	}

	langInstruction := languageInstructions[language]
	if langInstruction == "" {
		langInstruction = languageInstructions["en"]
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
  "calories": total calories for specified weight,
  "protein": total protein for specified weight,
  "fat": total fat for specified weight,
  "carbs": total carbs for specified weight
}`, langInstruction)

	userMessage := fmt.Sprintf("Food name: %s\nDescription: %s\nTotal weight: %.1fg\n\nProvide nutritional information for this food.", foodName, foodDescription, totalWeight)

	resp, err := s.openaiClient.CreateChatCompletion(
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

	return &resp, nil
}

// parseOpenAIResponse parses OpenAI response into FoodAnalysisResult
func (s *service) parseOpenAIResponse(response *openai.ChatCompletionResponse, totalWeight *float64) (*FoodAnalysisResult, error) {
	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no response from openai")
	}

	content := response.Choices[0].Message.Content

	// Try to parse as JSON
	var result FoodAnalysisResult
	err := json.Unmarshal([]byte(content), &result)
	if err != nil {
		// If parsing fails, try to extract JSON from markdown code block
		if strings.Contains(content, "```json") {
			start := strings.Index(content, "```json") + 7
			end := strings.LastIndex(content, "```")
			if end > start {
				jsonStr := content[start:end]
				err = json.Unmarshal([]byte(jsonStr), &result)
			}
		}
		if err != nil {
			return nil, fmt.Errorf("failed to parse openai response: %w", err)
		}
	}

	// Set flag indicating if user provided weight
	result.UserProvidedWeight = (totalWeight != nil)

	// If weight was not provided by user, AI should have estimated it
	// The result should already contain EstimatedWeight and WeightUnit from AI

	return &result, nil
}

// detectViolation checks if response indicates a violation
func (s *service) detectViolation(response *openai.ChatCompletionResponse) bool {
	if len(response.Choices) == 0 {
		return false
	}

	content := strings.ToLower(response.Choices[0].Message.Content)

	// Check for violation indicators
	return strings.Contains(content, `"violation": true`) ||
		strings.Contains(content, "not food-related") ||
		strings.Contains(content, "inappropriate")
}

// handleViolation creates violation record and bans user
func (s *service) handleViolation(ctx context.Context, userId string, logId int64, imageUrl, userPrompt, violationType, reason string) {
	// Create 7-day ban
	banUntil := time.Now().Add(7 * 24 * time.Hour)

	violation := Violation{
		UserId:          userId,
		AnalysisLogId:   &logId,
		ViolationType:   violationType,
		ViolationReason: reason,
		ImageUrl:        &imageUrl,
		UserPrompt:      &userPrompt,
		ActionTaken:     "temp_ban",
		BanUntil:        &banUntil,
		Reviewed:        false,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	createdViolation, err := s.repo.CreateViolation(ctx, violation)
	if err != nil {
		s.logger.Error("failed to create violation", "userId", userId, "error", err)
		return
	}

	// Set user ban
	err = s.repo.SetUserBan(ctx, userId, banUntil)
	if err != nil {
		s.logger.Error("failed to set user ban", "userId", userId, "error", err)
	}

	// Create admin notification
	metadata := fmt.Sprintf(`{"violationId": %d, "userId": "%s", "imageUrl": "%s"}`, createdViolation.Id, userId, imageUrl)
	notification := AdminNotification{
		NotificationType: "ai_violation",
		Title:            "AI Content Violation Detected",
		Message:          fmt.Sprintf("User %s uploaded inappropriate content. Banned until %s. Reason: %s", userId, banUntil.Format("2006-01-02"), reason),
		Severity:         "warning",
		UserId:           &userId,
		RelatedId:        &createdViolation.Id,
		Metadata:         &metadata,
		Read:             false,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	_, err = s.repo.CreateAdminNotification(ctx, notification)
	if err != nil {
		s.logger.Error("failed to create admin notification", "error", err)
	}

	s.logger.Warn("violation handled", "userId", userId, "violationId", createdViolation.Id, "banUntil", banUntil)
}

// calculateCost estimates API call cost based on tokens
func (s *service) calculateCost(promptTokens, completionTokens int) float64 {
	// GPT-4o pricing (as of 2024)
	// $5.00 / 1M input tokens
	// $15.00 / 1M output tokens
	promptCost := float64(promptTokens) * 5.0 / 1_000_000
	completionCost := float64(completionTokens) * 15.0 / 1_000_000
	return promptCost + completionCost
}

// updateLogWithError updates log with error status
func (s *service) updateLogWithError(ctx context.Context, logId int64, status string, err error) {
	errorMsg := err.Error()
	processingTime := 0
	_ = s.repo.UpdateAnalysisLog(ctx, AnalysisLog{
		Id:               logId,
		Status:           "error",
		ErrorMessage:     &errorMsg,
		ProcessingTimeMs: &processingTime,
		UpdatedAt:        time.Now(),
	})
}

// updateLogWithStatus updates log with custom status
func (s *service) updateLogWithStatus(ctx context.Context, logId int64, status, message string) {
	processingTime := 0
	_ = s.repo.UpdateAnalysisLog(ctx, AnalysisLog{
		Id:               logId,
		Status:           status,
		ErrorMessage:     &message,
		ProcessingTimeMs: &processingTime,
		UpdatedAt:        time.Now(),
	})
}
