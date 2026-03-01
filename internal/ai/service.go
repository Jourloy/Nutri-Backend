package ai

import (
	"bytes"
	"context"
	"encoding/base64"
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

	"github.com/jourloy/nutri-backend/internal/lib"
)

type Service interface {
	AnalyzeFoodImage(ctx context.Context, userId string, imageData []byte, totalWeight *float64, userPrompt string, language string) (*FoodAnalysisResult, error)
	AnalyzeFoodByText(ctx context.Context, userId string, foodName string, foodDescription string, totalWeight float64, language string) (*FoodAnalysisResult, error)
	CheckUserLimit(ctx context.Context, userId, requestType string) (*LimitCheckResult, error)
	GetUserAnalysisHistory(ctx context.Context, userId string, limit int) ([]AnalysisLog, error)
	ImproveText(ctx context.Context, userId string, html string) (string, error)
	GenerateArticle(ctx context.Context, userId string, topic string, description string, provider string) (*GeneratedArticle, error)
	GenerateRecipeDraft(ctx context.Context, userId string, titleRu, ingredientsRu, stepsRu string, imageData []byte, imageURL, provider string) (*GeneratedRecipeDraft, error)
}

type service struct {
	repo             Repository
	aiProvider       AIProvider
	aiProviderName   string
	fallbackProvider AIProvider
	fallbackName     string
	minioClient      *minio.Client
	logger           *log.Logger
}

func providerNameFromInstance(p AIProvider) string {
	switch p.(type) {
	case *OpenAIProvider:
		return "openai"
	case *PerplexityProvider:
		return "perplexity"
	default:
		return "unknown"
	}
}

func NewService(repo Repository) (Service, error) {
	// Initialize logger first
	logger := log.NewWithOptions(os.Stderr, log.Options{
		Prefix: "[ai-svc]",
		Level:  log.DebugLevel,
	})

	// Initialize AI provider
	aiProvider, err := GetProvider(logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize AI provider: %w", err)
	}

	logger.Info("AI provider initialized", "model", aiProvider.GetModelName())

	aiProviderName := providerNameFromInstance(aiProvider)

	fallbackProvider, err := GetFallbackProvider(logger, aiProvider)
	if err != nil {
		// Fallback is optional; log and continue.
		logger.Warn("failed to initialize AI fallback provider", "error", err)
	}
	fallbackName := providerNameFromInstance(fallbackProvider)

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
		repo:             repo,
		aiProvider:       aiProvider,
		aiProviderName:   aiProviderName,
		fallbackProvider: fallbackProvider,
		fallbackName:     fallbackName,
		minioClient:      minioClient,
		logger:           logger,
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
		ModelUsed:   s.aiProvider.GetModelName(),
		Status:      "pending",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	createdLog, err := s.repo.CreateAnalysisLog(ctx, analysisLog)
	if err != nil {
		return nil, fmt.Errorf("failed to create analysis log: %w", err)
	}

	// 4. Process and upload image to Minio
	imageUrl, imageBase64, err := s.processAndUploadImage(ctx, userId, imageData)
	if err != nil {
		s.updateLogWithError(ctx, createdLog.Id, "failed to upload image", err)
		return nil, fmt.Errorf("failed to upload image: %w", err)
	}
	createdLog.ImageUrl = imageUrl

	// 5. Call AI provider for image analysis
	response, err := s.aiProvider.AnalyzeImage(ctx, ImageAnalysisRequest{
		ImageURL:    imageUrl,
		ImageBase64: imageBase64,
		UserPrompt:  userPrompt,
		TotalWeight: totalWeight,
		Language:    language,
	})
	if err != nil {
		s.updateLogWithError(ctx, createdLog.Id, "ai provider error", err)
		return nil, fmt.Errorf("ai provider error: %w", err)
	}

	// 6. Check for content moderation flags
	if response.IsViolation {
		s.handleViolation(ctx, userId, createdLog.Id, imageUrl, userPrompt, "off_topic", response.ViolationReason)
		s.updateLogWithStatus(ctx, createdLog.Id, "moderated", "Content moderated")
		return nil, fmt.Errorf("content moderated: %s", response.ViolationReason)
	}

	// 7. Convert response to FoodAnalysisResult
	result := &FoodAnalysisResult{
		ProductName:        response.ProductName,
		Confidence:         response.Confidence,
		Explanation:        response.Explanation,
		BasicCalories:      response.BasicCalories,
		BasicProtein:       response.BasicProtein,
		BasicFat:           response.BasicFat,
		BasicCarbs:         response.BasicCarbs,
		BasicFiber:         response.BasicFiber,
		BasicCholesterol:   response.BasicCholesterol,
		Calories:           response.Calories,
		Protein:            response.Protein,
		Fat:                response.Fat,
		Carbs:              response.Carbs,
		Fiber:              response.Fiber,
		Cholesterol:        response.Cholesterol,
		EstimatedWeight:    response.EstimatedWeight,
		WeightUnit:         response.WeightUnit,
		UserProvidedWeight: totalWeight != nil,
	}

	// 8. Calculate costs
	tokensPrompt := response.PromptTokens
	tokensCompletion := response.CompletionTokens
	estimatedCost := s.aiProvider.CalculateCost(tokensPrompt, tokensCompletion)

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
		ModelUsed:   s.aiProvider.GetModelName(),
		Status:      "pending",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	createdLog, err := s.repo.CreateAnalysisLog(ctx, analysisLog)
	if err != nil {
		return nil, fmt.Errorf("failed to create analysis log: %w", err)
	}

	// 4. Call AI provider for text analysis
	response, err := s.aiProvider.AnalyzeText(ctx, TextAnalysisRequest{
		FoodName:        foodName,
		FoodDescription: foodDescription,
		TotalWeight:     totalWeight,
		Language:        language,
	})
	if err != nil {
		s.updateLogWithError(ctx, createdLog.Id, "ai provider error", err)
		return nil, fmt.Errorf("ai provider error: %w", err)
	}

	// 5. Convert response to FoodAnalysisResult
	result := &FoodAnalysisResult{
		ProductName:        response.ProductName,
		Confidence:         response.Confidence,
		Explanation:        response.Explanation,
		BasicCalories:      response.BasicCalories,
		BasicProtein:       response.BasicProtein,
		BasicFat:           response.BasicFat,
		BasicCarbs:         response.BasicCarbs,
		BasicFiber:         response.BasicFiber,
		BasicCholesterol:   response.BasicCholesterol,
		Calories:           response.Calories,
		Protein:            response.Protein,
		Fat:                response.Fat,
		Carbs:              response.Carbs,
		Fiber:              response.Fiber,
		Cholesterol:        response.Cholesterol,
		EstimatedWeight:    response.EstimatedWeight,
		WeightUnit:         response.WeightUnit,
		UserProvidedWeight: true,
	}

	// 6. Calculate costs
	tokensPrompt := response.PromptTokens
	tokensCompletion := response.CompletionTokens
	estimatedCost := s.aiProvider.CalculateCost(tokensPrompt, tokensCompletion)

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

// ImproveText improves the given HTML without changing its meaning (admin tool).
func (s *service) ImproveText(ctx context.Context, userId string, html string) (string, error) {
	html = strings.TrimSpace(html)
	if html == "" {
		return "", fmt.Errorf("html is required")
	}
	if len(html) > 50_000 {
		return "", fmt.Errorf("html is too long")
	}
	return s.aiProvider.ImproveText(ctx, html)
}

func (s *service) selectGenerationProvider(provider string) (AIProvider, error) {
	p := strings.ToLower(strings.TrimSpace(provider))
	if p == "" || p == "auto" {
		return s.aiProvider, nil
	}

	if p == s.aiProviderName {
		return s.aiProvider, nil
	}
	if s.fallbackProvider != nil && p == s.fallbackName {
		return s.fallbackProvider, nil
	}

	switch p {
	case "openai", "perplexity":
		return nil, fmt.Errorf("provider %q is not available", p)
	default:
		return nil, fmt.Errorf("unknown provider %q", p)
	}
}

// GenerateArticle generates a full RU+EN blog article based on topic/description (admin tool).
func (s *service) GenerateArticle(ctx context.Context, userId string, topic string, description string, provider string) (*GeneratedArticle, error) {
	topic = strings.TrimSpace(topic)
	description = strings.TrimSpace(description)

	if topic == "" {
		return nil, fmt.Errorf("topic is required")
	}
	if description == "" {
		return nil, fmt.Errorf("description is required")
	}
	if len(topic) > 200 {
		return nil, fmt.Errorf("topic is too long")
	}
	if len(description) > 2000 {
		return nil, fmt.Errorf("description is too long")
	}

	p, err := s.selectGenerationProvider(provider)
	if err != nil {
		return nil, err
	}

	article, err := p.GenerateArticle(ctx, GenerateArticleRequest{Topic: topic, Description: description, Provider: provider})
	if err != nil {
		return nil, err
	}
	if article == nil {
		return nil, fmt.Errorf("empty article")
	}
	if article.TitleRu == "" || article.TitleEn == "" || article.ContentRu == "" || article.ContentEn == "" {
		return nil, fmt.Errorf("incomplete article generated")
	}

	// Normalize fields defensively (models sometimes append "(N символов)").
	article.PreviewTextRu = StripTrailingCharCount(article.PreviewTextRu)
	article.PreviewTextEn = StripTrailingCharCount(article.PreviewTextEn)
	article.MetaDescriptionRu = StripTrailingCharCount(article.MetaDescriptionRu)
	article.MetaDescriptionEn = StripTrailingCharCount(article.MetaDescriptionEn)
	article.ContentRu = StripInlineNumericCitations(article.ContentRu)
	article.ContentEn = StripInlineNumericCitations(article.ContentEn)
	article.ContentRu = EnsureAtLeastOneImageMarker(article.ContentRu, "")
	article.ContentEn = EnsureAtLeastOneImageMarker(article.ContentEn, "")
	article.Sources = NormalizeSourceURLs(article.Sources)
	if NormalizeGeneratedArticleLanguages(article) && s.logger != nil {
		s.logger.Warn(
			"GenerateArticle detected swapped RU/EN fields and normalized response",
			"provider",
			strings.ToLower(strings.TrimSpace(provider)),
		)
	}

	ruWords := ApproxWordCountFromHTML(article.ContentRu)
	enWords := ApproxWordCountFromHTML(article.ContentEn)
	if s.logger != nil && (ruWords < 900 || enWords < 900) {
		s.logger.Warn("GenerateArticle produced short content", "provider", strings.ToLower(strings.TrimSpace(provider)), "ru_words", ruWords, "en_words", enWords)
	}

	return article, nil
}

func (s *service) GenerateRecipeDraft(
	ctx context.Context,
	userId string,
	titleRu, ingredientsRu, stepsRu string,
	imageData []byte,
	imageURL, provider string,
) (*GeneratedRecipeDraft, error) {
	titleRu = strings.TrimSpace(titleRu)
	ingredientsRu = strings.TrimSpace(ingredientsRu)
	stepsRu = strings.TrimSpace(stepsRu)
	imageURL = strings.TrimSpace(imageURL)
	provider = strings.ToLower(strings.TrimSpace(provider))

	if titleRu == "" {
		return nil, fmt.Errorf("titleRu is required")
	}
	if ingredientsRu == "" {
		return nil, fmt.Errorf("ingredientsRu is required")
	}
	if stepsRu == "" {
		return nil, fmt.Errorf("stepsRu is required")
	}
	if len(titleRu) > 500 {
		return nil, fmt.Errorf("titleRu is too long")
	}
	if len(ingredientsRu) > 10_000 {
		return nil, fmt.Errorf("ingredientsRu is too long")
	}
	if len(stepsRu) > 20_000 {
		return nil, fmt.Errorf("stepsRu is too long")
	}
	if len(imageData) == 0 && imageURL == "" {
		return nil, fmt.Errorf("image or imageUrl is required")
	}

	var imageBase64 string
	if len(imageData) > 0 {
		uploadedImageURL, encodedImage, err := s.processAndUploadImage(ctx, userId, imageData)
		if err != nil {
			return nil, fmt.Errorf("failed to process image: %w", err)
		}
		imageURL = uploadedImageURL
		imageBase64 = encodedImage
	}

	selectedProvider, err := s.selectGenerationProvider(provider)
	if err != nil {
		return nil, err
	}

	draft, err := selectedProvider.GenerateRecipeDraft(ctx, GenerateRecipeDraftRequest{
		TitleRu:       titleRu,
		IngredientsRu: ingredientsRu,
		StepsRu:       stepsRu,
		ImageURL:      imageURL,
		ImageBase64:   imageBase64,
		Provider:      provider,
	})
	if err != nil {
		return nil, err
	}

	draft = NormalizeGeneratedRecipeDraft(draft)
	if draft == nil {
		return nil, fmt.Errorf("empty recipe draft")
	}
	if draft.TitleRu == "" {
		draft.TitleRu = titleRu
	}
	if draft.TitleEn == "" {
		return nil, fmt.Errorf("incomplete recipe draft generated")
	}
	if len(draft.Ingredients) == 0 || len(draft.Steps) == 0 {
		return nil, fmt.Errorf("incomplete recipe draft generated")
	}

	return draft, nil
}

// processAndUploadImage processes image (resize, compress) and uploads to Minio
// Returns: presigned URL for storage, base64 encoded image for AI providers
func (s *service) processAndUploadImage(ctx context.Context, userId string, imageData []byte) (string, string, error) {
	// Decode original image
	img, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return "", "", fmt.Errorf("failed to decode image: %w", err)
	}

	// Resize to max 800px width for cost savings
	resized := resize.Resize(800, 0, img, resize.Lanczos3)

	// Encode with lower quality (60%) for storage savings
	var buf bytes.Buffer
	err = jpeg.Encode(&buf, resized, &jpeg.Options{Quality: 60})
	if err != nil {
		return "", "", fmt.Errorf("failed to encode image: %w", err)
	}

	// Get base64 encoded image for AI providers
	imageBase64 := base64.StdEncoding.EncodeToString(buf.Bytes())

	// Generate unique filename
	filename := fmt.Sprintf("%s/%s_%d.jpg", userId, uuid.New().String(), time.Now().Unix())

	// Upload to Minio
	_, err = s.minioClient.PutObject(
		ctx,
		lib.Config.MinioBucketName,
		filename,
		bytes.NewReader(buf.Bytes()),
		int64(buf.Len()),
		minio.PutObjectOptions{
			ContentType: "image/jpeg",
		},
	)
	if err != nil {
		return "", "", fmt.Errorf("failed to upload to minio: %w", err)
	}

	// Generate presigned URL (valid for 7 days for moderation review)
	presignedURL, err := s.minioClient.PresignedGetObject(ctx, lib.Config.MinioBucketName, filename, 7*24*time.Hour, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate presigned url: %w", err)
	}

	return presignedURL.String(), imageBase64, nil
}

// callOpenAIVision calls OpenAI Vision API
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
