package blog

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	htmlrenderer "github.com/yuin/goldmark/renderer/html"

	"github.com/jourloy/nutri-backend/internal/ai"
	"github.com/jourloy/nutri-backend/internal/lib"
)

const blogBucketName = "nutri-blog-images"

const (
	maxPrepareTitleLength       = 220
	maxPrepareDescriptionLength = 2000
	maxPrepareMarkdownLength    = 120000
)

var (
	ruToLatReplacer = strings.NewReplacer(
		"а", "a", "б", "b", "в", "v", "г", "g", "д", "d", "е", "e", "ё", "yo",
		"ж", "zh", "з", "z", "и", "i", "й", "y", "к", "k", "л", "l", "м", "m",
		"н", "n", "о", "o", "п", "p", "р", "r", "с", "s", "т", "t", "у", "u",
		"ф", "f", "х", "h", "ц", "ts", "ч", "ch", "ш", "sh", "щ", "sch", "ъ", "",
		"ы", "y", "ь", "", "э", "e", "ю", "yu", "я", "ya",
	)
	blogMarkdownParser = goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithRendererOptions(htmlrenderer.WithUnsafe()),
	)
)

type Service interface {
	// Categories (Admin)
	CreateCategory(ctx context.Context, c CategoryCreate) (*Category, error)
	UpdateCategory(ctx context.Context, c CategoryUpdate) (*Category, error)
	DeleteCategory(ctx context.Context, id int64) error
	GetAllCategories(ctx context.Context) ([]Category, error)

	// Tags (Admin)
	CreateTag(ctx context.Context, t TagCreate) (*Tag, error)
	UpdateTag(ctx context.Context, t TagUpdate) (*Tag, error)
	DeleteTag(ctx context.Context, id int64) error
	GetAllTags(ctx context.Context) ([]Tag, error)

	// Articles (Admin)
	CreateArticle(ctx context.Context, a ArticleCreate) (*Article, error)
	PrepareArticle(ctx context.Context, req PrepareArticleRequest) (*PrepareArticleResponse, error)
	UpdateArticle(ctx context.Context, a ArticleUpdate) (*Article, error)
	DeleteArticle(ctx context.Context, id int64) error
	GetArticleById(ctx context.Context, id int64) (*Article, error)
	GetAllArticles(ctx context.Context, params ArticleListParams) (*ArticleListResponse, error)

	// Articles (Public) - with access control
	GetPublicArticles(ctx context.Context, params ArticleListParams, viewer ViewerAccess) (*ArticlePublicListResponse, error)
	GetPublicArticleBySlug(ctx context.Context, slug string, viewer ViewerAccess) (*ArticlePublic, error)
	TrackView(ctx context.Context, articleId int64) error

	// Feedback
	SubmitFeedback(ctx context.Context, f FeedbackCreate) (*Feedback, error)
	GetFeedbackStats(ctx context.Context, articleId int64) (*FeedbackStats, error)
	HasUserFeedback(ctx context.Context, articleId int64, userId *string, sessionId *string) (bool, error)

	// Image Upload
	UploadImage(ctx context.Context, imageData []byte, filename string) (string, error)
}

type service struct {
	repo             Repository
	minioClient      *minio.Client
	logger           *log.Logger
	aiProvider       ai.AIProvider
	aiProviderName   string
	fallbackProvider ai.AIProvider
	fallbackName     string
}

func NewService(repo Repository) (Service, error) {
	logger := log.NewWithOptions(os.Stderr, log.Options{
		Prefix: "[blog-svc]",
		Level:  log.DebugLevel,
	})

	// Parse Minio endpoint
	endpoint := lib.Config.MinioEndpoint
	useSSL := lib.Config.MinioUseSSL

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

	// Ensure bucket exists
	err = minioClient.MakeBucket(context.Background(), blogBucketName, minio.MakeBucketOptions{})
	if err != nil {
		exists, errBucketExists := minioClient.BucketExists(context.Background(), blogBucketName)
		if errBucketExists != nil {
			logger.Warn("failed to check bucket existence", "bucket", blogBucketName, "error", errBucketExists)
		}
		if !exists {
			logger.Warn("bucket does not exist and could not be created", "bucket", blogBucketName, "error", err)
		} else {
			logger.Debug("bucket already exists", "bucket", blogBucketName)
		}
	} else {
		logger.Info("bucket created successfully", "bucket", blogBucketName)
	}

	articleProvider, err := ai.GetProvider(logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize AI provider for blog preparation: %w", err)
	}
	articleProviderName := "unknown"
	switch articleProvider.(type) {
	case *ai.OpenAIProvider:
		articleProviderName = "openai"
	case *ai.PerplexityProvider:
		articleProviderName = "perplexity"
	}

	fallbackProvider, err := ai.GetFallbackProvider(logger, articleProvider)
	if err != nil {
		logger.Warn("failed to initialize AI fallback provider for blog preparation", "error", err)
	}
	fallbackName := "unknown"
	switch fallbackProvider.(type) {
	case *ai.OpenAIProvider:
		fallbackName = "openai"
	case *ai.PerplexityProvider:
		fallbackName = "perplexity"
	}

	return &service{
		repo:             repo,
		minioClient:      minioClient,
		logger:           logger,
		aiProvider:       articleProvider,
		aiProviderName:   articleProviderName,
		fallbackProvider: fallbackProvider,
		fallbackName:     fallbackName,
	}, nil
}

// ===== Categories =====

func (s *service) CreateCategory(ctx context.Context, c CategoryCreate) (*Category, error) {
	return s.repo.CreateCategory(ctx, c)
}

func (s *service) UpdateCategory(ctx context.Context, c CategoryUpdate) (*Category, error) {
	return s.repo.UpdateCategory(ctx, c)
}

func (s *service) DeleteCategory(ctx context.Context, id int64) error {
	return s.repo.DeleteCategory(ctx, id)
}

func (s *service) GetAllCategories(ctx context.Context) ([]Category, error) {
	return s.repo.GetAllCategories(ctx)
}

// ===== Tags =====

func (s *service) CreateTag(ctx context.Context, t TagCreate) (*Tag, error) {
	return s.repo.CreateTag(ctx, t)
}

func (s *service) UpdateTag(ctx context.Context, t TagUpdate) (*Tag, error) {
	return s.repo.UpdateTag(ctx, t)
}

func (s *service) DeleteTag(ctx context.Context, id int64) error {
	return s.repo.DeleteTag(ctx, id)
}

func (s *service) GetAllTags(ctx context.Context) ([]Tag, error) {
	return s.repo.GetAllTags(ctx)
}

// ===== Articles (Admin) =====

func (s *service) CreateArticle(ctx context.Context, a ArticleCreate) (*Article, error) {
	a.Sources = normalizeSourceURLs(a.Sources)

	article, err := s.repo.CreateArticle(ctx, a)
	if err != nil {
		return nil, err
	}

	// Set article tags
	if len(a.TagIds) > 0 {
		if err := s.repo.SetArticleTags(ctx, article.Id, a.TagIds); err != nil {
			s.logger.Error("failed to set article tags", "articleId", article.Id, "error", err)
		}
	}

	// Set published_at only for publishable statuses.
	if isPublishableStatus(a.Status) {
		if err := s.repo.SetPublishedAt(ctx, article.Id); err != nil {
			s.logger.Error("failed to set published_at", "articleId", article.Id, "error", err)
		}
	}

	// Load full article with relations
	return s.loadArticleWithRelations(ctx, article)
}

func (s *service) PrepareArticle(ctx context.Context, req PrepareArticleRequest) (*PrepareArticleResponse, error) {
	titleRu := strings.TrimSpace(req.TitleRu)
	descriptionRu := strings.TrimSpace(req.DescriptionRu)
	contentMarkdownRu := strings.TrimSpace(req.ContentMarkdown)

	if titleRu == "" {
		return nil, fmt.Errorf("titleRu is required")
	}
	if descriptionRu == "" {
		return nil, fmt.Errorf("descriptionRu is required")
	}
	if contentMarkdownRu == "" {
		return nil, fmt.Errorf("contentMarkdownRu is required")
	}
	if len(titleRu) > maxPrepareTitleLength {
		return nil, fmt.Errorf("titleRu is too long")
	}
	if len(descriptionRu) > maxPrepareDescriptionLength {
		return nil, fmt.Errorf("descriptionRu is too long")
	}
	if len(contentMarkdownRu) > maxPrepareMarkdownLength {
		return nil, fmt.Errorf("contentMarkdownRu is too long")
	}

	contentRu, err := markdownToHTML(contentMarkdownRu)
	if err != nil {
		return nil, fmt.Errorf("failed to convert markdown to html: %w", err)
	}
	if strings.TrimSpace(contentRu) == "" {
		return nil, fmt.Errorf("contentMarkdownRu produced empty html")
	}

	generated, err := s.generatePreparedArticle(ctx, titleRu, descriptionRu, contentRu)
	if err != nil {
		return nil, err
	}
	if generated == nil {
		return nil, fmt.Errorf("failed to prepare article")
	}

	if ai.NormalizeGeneratedArticleLanguages(generated) && s.logger != nil {
		s.logger.Warn("prepare article detected swapped RU/EN fields and normalized response")
	}

	generated.PreviewTextRu = ai.StripTrailingCharCount(generated.PreviewTextRu)
	generated.PreviewTextEn = ai.StripTrailingCharCount(generated.PreviewTextEn)
	generated.MetaDescriptionRu = ai.StripTrailingCharCount(generated.MetaDescriptionRu)
	generated.MetaDescriptionEn = ai.StripTrailingCharCount(generated.MetaDescriptionEn)
	generated.ContentEn = ai.StripInlineNumericCitations(generated.ContentEn)
	generated.ContentEn = ai.EnsureAtLeastOneImageMarker(generated.ContentEn, "")
	generated.Sources = ai.NormalizeSourceURLs(generated.Sources)

	titleEn := strings.TrimSpace(generated.TitleEn)
	if titleEn == "" {
		return nil, fmt.Errorf("prepared titleEn is empty")
	}
	contentEn := strings.TrimSpace(generated.ContentEn)
	if contentEn == "" {
		return nil, fmt.Errorf("prepared contentEn is empty")
	}

	previewTextRu := firstNonEmpty(strings.TrimSpace(generated.PreviewTextRu), descriptionRu)
	previewTextEn := firstNonEmpty(strings.TrimSpace(generated.PreviewTextEn), strings.TrimSpace(generated.PreviewTextRu))
	metaDescriptionRu := firstNonEmpty(strings.TrimSpace(generated.MetaDescriptionRu), previewTextRu)
	metaDescriptionEn := firstNonEmpty(strings.TrimSpace(generated.MetaDescriptionEn), previewTextEn)

	slug, err := s.generateUniqueSlug(ctx, titleRu)
	if err != nil {
		return nil, fmt.Errorf("failed to generate slug: %w", err)
	}

	return &PrepareArticleResponse{
		Slug:              slug,
		TitleRu:           titleRu,
		TitleEn:           titleEn,
		ContentRu:         contentRu,
		ContentEn:         contentEn,
		PreviewTextRu:     previewTextRu,
		PreviewTextEn:     previewTextEn,
		MetaDescriptionRu: metaDescriptionRu,
		MetaDescriptionEn: metaDescriptionEn,
		Sources:           normalizeSourceURLs(generated.Sources),
	}, nil
}

func (s *service) UpdateArticle(ctx context.Context, a ArticleUpdate) (*Article, error) {
	// Get current article to check status change
	current, err := s.repo.GetArticleById(ctx, a.Id)
	if err != nil {
		return nil, err
	}

	a.Sources = normalizeSourceURLs(a.Sources)

	article, err := s.repo.UpdateArticle(ctx, a)
	if err != nil {
		return nil, err
	}

	// Update tags
	if err := s.repo.SetArticleTags(ctx, article.Id, a.TagIds); err != nil {
		s.logger.Error("failed to update article tags", "articleId", article.Id, "error", err)
	}

	// Set published_at only when transitioning into a publishable status.
	if !isPublishableStatus(current.Status) && isPublishableStatus(a.Status) {
		if err := s.repo.SetPublishedAt(ctx, article.Id); err != nil {
			s.logger.Error("failed to set published_at", "articleId", article.Id, "error", err)
		}
	}

	return s.loadArticleWithRelations(ctx, article)
}

func (s *service) DeleteArticle(ctx context.Context, id int64) error {
	return s.repo.DeleteArticle(ctx, id)
}

func (s *service) GetArticleById(ctx context.Context, id int64) (*Article, error) {
	article, err := s.repo.GetArticleById(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.loadArticleWithRelations(ctx, article)
}

func (s *service) GetAllArticles(ctx context.Context, params ArticleListParams) (*ArticleListResponse, error) {
	response, err := s.repo.GetArticles(ctx, params, true)
	if err != nil {
		return nil, err
	}

	// Load relations for each article
	for i := range response.Articles {
		s.loadArticleRelationsInPlace(ctx, &response.Articles[i])
	}

	return response, nil
}

// ===== Articles (Public) =====

func (s *service) GetPublicArticles(ctx context.Context, params ArticleListParams, viewer ViewerAccess) (*ArticlePublicListResponse, error) {
	params.AllowedStatuses = allowedStatusesForViewer(viewer)

	response, err := s.repo.GetArticles(ctx, params, false)
	if err != nil {
		return nil, err
	}

	// Convert to public response.
	publicArticles := make([]ArticlePublic, 0, len(response.Articles))
	for i := range response.Articles {
		s.loadArticleRelationsInPlace(ctx, &response.Articles[i])
		publicArticles = append(publicArticles, response.Articles[i].ToPublic())
	}

	return &ArticlePublicListResponse{
		Articles:   publicArticles,
		Total:      response.Total,
		Page:       response.Page,
		PerPage:    response.PerPage,
		TotalPages: response.TotalPages,
	}, nil
}

func (s *service) GetPublicArticleBySlug(ctx context.Context, slug string, viewer ViewerAccess) (*ArticlePublic, error) {
	article, err := s.repo.GetArticleBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	if !CanAccessArticle(article.Status, viewer) {
		return nil, fmt.Errorf("access denied")
	}

	article, err = s.loadArticleWithRelations(ctx, article)
	if err != nil {
		return nil, err
	}

	public := article.ToPublic()
	return &public, nil
}

func (s *service) TrackView(ctx context.Context, articleId int64) error {
	return s.repo.IncrementViewCount(ctx, articleId)
}

// ===== Feedback =====

func (s *service) SubmitFeedback(ctx context.Context, f FeedbackCreate) (*Feedback, error) {
	// Check if user already submitted feedback
	if f.UserId != nil && *f.UserId != "" {
		existing, err := s.repo.GetFeedbackByUser(ctx, f.ArticleId, *f.UserId)
		if err == nil && existing != nil {
			return nil, fmt.Errorf("feedback already submitted")
		}
		if err != nil && err != sql.ErrNoRows {
			return nil, err
		}
	} else if f.SessionId != nil && *f.SessionId != "" {
		existing, err := s.repo.GetFeedbackBySession(ctx, f.ArticleId, *f.SessionId)
		if err == nil && existing != nil {
			return nil, fmt.Errorf("feedback already submitted")
		}
		if err != nil && err != sql.ErrNoRows {
			return nil, err
		}
	}

	return s.repo.CreateFeedback(ctx, f)
}

func (s *service) GetFeedbackStats(ctx context.Context, articleId int64) (*FeedbackStats, error) {
	return s.repo.GetFeedbackStats(ctx, articleId)
}

func (s *service) HasUserFeedback(ctx context.Context, articleId int64, userId *string, sessionId *string) (bool, error) {
	if userId != nil && *userId != "" {
		existing, err := s.repo.GetFeedbackByUser(ctx, articleId, *userId)
		if err == sql.ErrNoRows {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		return existing != nil, nil
	}

	if sessionId != nil && *sessionId != "" {
		existing, err := s.repo.GetFeedbackBySession(ctx, articleId, *sessionId)
		if err == sql.ErrNoRows {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		return existing != nil, nil
	}

	return false, nil
}

// ===== Image Upload =====

func (s *service) UploadImage(ctx context.Context, imageData []byte, filename string) (string, error) {
	// Generate unique filename
	ext := ""
	if idx := strings.LastIndex(filename, "."); idx != -1 {
		ext = filename[idx:]
	}
	if ext == "" {
		ext = ".jpg"
	}

	objectName := fmt.Sprintf("%s/%s%s", time.Now().Format("2006/01"), uuid.New().String(), ext)

	// Determine content type
	contentType := "image/jpeg"
	switch strings.ToLower(ext) {
	case ".png":
		contentType = "image/png"
	case ".gif":
		contentType = "image/gif"
	case ".webp":
		contentType = "image/webp"
	}

	// Upload to MinIO
	_, err := s.minioClient.PutObject(ctx, blogBucketName, objectName, bytes.NewReader(imageData), int64(len(imageData)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload image: %w", err)
	}

	// Generate public URL
	// If using public bucket, construct direct URL
	endpoint := lib.Config.MinioEndpoint
	if !strings.HasPrefix(endpoint, "http") {
		if lib.Config.MinioUseSSL {
			endpoint = "https://" + endpoint
		} else {
			endpoint = "http://" + endpoint
		}
	}

	imageUrl := fmt.Sprintf("%s/%s/%s", endpoint, blogBucketName, objectName)

	return imageUrl, nil
}

// ===== Helpers =====

func (s *service) generatePreparedArticle(
	ctx context.Context,
	titleRu string,
	descriptionRu string,
	contentRu string,
) (*ai.GeneratedArticle, error) {
	req := ai.GenerateArticleRequest{
		TitleRu:     titleRu,
		Topic:       titleRu,
		Description: descriptionRu,
		ContentRu:   contentRu,
		Provider:    "auto",
	}

	primary, err := s.aiProvider.GenerateArticle(ctx, req)
	if err == nil && primary != nil {
		return primary, nil
	}

	if s.logger != nil && err != nil {
		s.logger.Warn(
			"primary article preparation provider failed",
			"provider",
			s.aiProviderName,
			"error",
			err,
		)
	}

	if s.fallbackProvider == nil {
		if err != nil {
			return nil, fmt.Errorf("failed to prepare article with provider %s: %w", s.aiProviderName, err)
		}
		return nil, fmt.Errorf("failed to prepare article")
	}

	fallbackReq := req
	fallbackReq.Provider = s.fallbackName
	fallback, fallbackErr := s.fallbackProvider.GenerateArticle(ctx, fallbackReq)
	if fallbackErr != nil {
		return nil, fmt.Errorf(
			"failed to prepare article with providers %s/%s: %w / %v",
			s.aiProviderName,
			s.fallbackName,
			err,
			fallbackErr,
		)
	}
	return fallback, nil
}

func markdownToHTML(markdown string) (string, error) {
	var out bytes.Buffer
	if err := blogMarkdownParser.Convert([]byte(markdown), &out); err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}

func slugifyTitle(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = ruToLatReplacer.Replace(value)

	var b strings.Builder
	prevDash := false

	for _, r := range value {
		isLatin := r >= 'a' && r <= 'z'
		isDigit := r >= '0' && r <= '9'

		if isLatin || isDigit {
			b.WriteRune(r)
			prevDash = false
			continue
		}

		if !prevDash {
			b.WriteByte('-')
			prevDash = true
		}
	}

	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		return "article"
	}
	return slug
}

func (s *service) generateUniqueSlug(ctx context.Context, title string) (string, error) {
	base := slugifyTitle(title)
	slug := base

	for index := 2; index < 10000; index++ {
		exists, err := s.repo.ArticleSlugExists(ctx, slug)
		if err != nil {
			return "", err
		}
		if !exists {
			return slug, nil
		}
		slug = base + "-" + strconv.Itoa(index)
	}

	return "", fmt.Errorf("unable to generate unique slug")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func normalizeSourceURLs(sources []string) []string {
	if len(sources) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(sources))
	seen := make(map[string]struct{}, len(sources))

	for _, raw := range sources {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}

		parsed, err := url.Parse(raw)
		if err != nil || !parsed.IsAbs() || parsed.Host == "" {
			continue
		}

		scheme := strings.ToLower(parsed.Scheme)
		if scheme != "http" && scheme != "https" {
			continue
		}
		parsed.Scheme = scheme
		parsed.Host = strings.ToLower(parsed.Host)

		cleaned := strings.TrimSpace(parsed.String())
		if cleaned == "" {
			continue
		}

		key := strings.ToLower(cleaned)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, cleaned)
	}

	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func (s *service) loadArticleWithRelations(ctx context.Context, article *Article) (*Article, error) {
	s.loadArticleRelationsInPlace(ctx, article)
	return article, nil
}

func (s *service) loadArticleRelationsInPlace(ctx context.Context, article *Article) {
	// Load category
	if article.CategoryId != nil {
		cat, err := s.repo.GetCategoryById(ctx, *article.CategoryId)
		if err == nil {
			article.Category = cat
		}
	}

	// Load tags
	tags, err := s.repo.GetTagsByArticleId(ctx, article.Id)
	if err == nil {
		article.Tags = tags
	} else {
		article.Tags = []Tag{}
	}
}

func isPublishableStatus(status string) bool {
	switch status {
	case "authorized", "paid", "public":
		return true
	default:
		return false
	}
}

func hasPaidPlanCode(planCode string) bool {
	normalized := strings.TrimSpace(strings.ToUpper(planCode))
	return normalized != "" && normalized != "START"
}

func allowedStatusesForViewer(viewer ViewerAccess) []string {
	if viewer.IsAdmin {
		return []string{"preview", "authorized", "paid", "public"}
	}

	allowed := []string{"public"}
	if viewer.IsAuthenticated {
		allowed = append(allowed, "authorized")
		if hasPaidPlanCode(viewer.PlanCode) {
			allowed = append(allowed, "paid")
		}
	}

	return allowed
}

// CanAccessArticle checks if a user can access an article based on status and viewer access context.
func CanAccessArticle(status string, viewer ViewerAccess) bool {
	switch status {
	case "public":
		return true
	case "preview":
		return viewer.IsAdmin
	case "authorized":
		return viewer.IsAuthenticated || viewer.IsAdmin
	case "paid":
		return viewer.IsAdmin || (viewer.IsAuthenticated && hasPaidPlanCode(viewer.PlanCode))
	case "draft":
		return false
	default:
		return false
	}
}
