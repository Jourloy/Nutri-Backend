package blog

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/go-chi/chi/v5"

	"github.com/jourloy/nutri-backend/internal/auth"
	"github.com/jourloy/nutri-backend/internal/subscription"
)

var (
	logger = log.NewWithOptions(os.Stderr, log.Options{
		Prefix: "[blog]",
		Level:  log.DebugLevel,
	})
)

type Controller struct {
	service Service
}

func NewController() (*Controller, error) {
	repo := NewRepository()
	svc, err := NewServiceFromConfig(repo)
	if err != nil {
		return nil, err
	}

	return &Controller{
		service: svc,
	}, nil
}

func (c *Controller) RegisterRoutes(router chi.Router) {
	router.Route("/blog", func(r chi.Router) {
		// Public endpoints
		r.Get("/articles", c.GetPublicArticles)
		r.Get("/articles/{slug}", c.GetPublicArticle)
		r.Get("/categories", c.GetCategories)
		r.Get("/tags", c.GetTags)
		r.Post("/articles/{id}/feedback", c.SubmitFeedback)
		r.Get("/articles/{id}/feedback/stats", c.GetFeedbackStats)

		// Admin endpoints
		r.Group(func(r chi.Router) {
			r.Use(requireAdmin)

			// Articles
			r.Post("/admin/articles/prepare", c.PrepareArticle)
			r.Post("/admin/articles", c.CreateArticle)
			r.Put("/admin/articles/{id}", c.UpdateArticle)
			r.Delete("/admin/articles/{id}", c.DeleteArticle)
			r.Get("/admin/articles", c.GetAllArticles)
			r.Get("/admin/articles/{id}", c.GetArticleById)

			// Categories
			r.Post("/admin/categories", c.CreateCategory)
			r.Put("/admin/categories/{id}", c.UpdateCategory)
			r.Delete("/admin/categories/{id}", c.DeleteCategory)

			// Tags
			r.Post("/admin/tags", c.CreateTag)
			r.Put("/admin/tags/{id}", c.UpdateTag)
			r.Delete("/admin/tags/{id}", c.DeleteTag)

			// Image Upload
			r.Post("/admin/upload", c.UploadImage)
		})
	})

	logger.Info("╔═════ Blog")
	logger.Info("║    GET /blog/articles (get public articles)")
	logger.Info("║    GET /blog/articles/{slug} (get article by slug)")
	logger.Info("║    GET /blog/categories (get all categories)")
	logger.Info("║    GET /blog/tags (get all tags)")
	logger.Info("║   POST /blog/articles/{id}/feedback (submit feedback)")
	logger.Info("║    GET /blog/articles/{id}/feedback/stats (get feedback stats)")
	logger.Info("║   POST /blog/admin/articles/prepare (admin: prepare article from RU markdown)")
	logger.Info("║   POST /blog/admin/articles (admin: create article)")
	logger.Info("║    PUT /blog/admin/articles/{id} (admin: update article)")
	logger.Info("║ DELETE /blog/admin/articles/{id} (admin: delete article)")
	logger.Info("║    GET /blog/admin/articles (admin: get all articles)")
	logger.Info("║    GET /blog/admin/articles/{id} (admin: get article by id)")
	logger.Info("║   POST /blog/admin/categories (admin: create category)")
	logger.Info("║    PUT /blog/admin/categories/{id} (admin: update category)")
	logger.Info("║ DELETE /blog/admin/categories/{id} (admin: delete category)")
	logger.Info("║   POST /blog/admin/tags (admin: create tag)")
	logger.Info("║    PUT /blog/admin/tags/{id} (admin: update tag)")
	logger.Info("║ DELETE /blog/admin/tags/{id} (admin: delete tag)")
	logger.Info("║   POST /blog/admin/upload (admin: upload image)")
	logger.Info("╚═════")
}

// Middleware to require admin access
func requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, ok := auth.UserFromContext(r.Context())
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		if !u.IsAdmin {
			http.Error(w, "forbidden: admin access required", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// ===== Helper Functions =====

func getViewerAccess(r *http.Request) ViewerAccess {
	access := ViewerAccess{}

	if u, ok := auth.UserFromContext(r.Context()); ok {
		access.IsAuthenticated = true
		access.IsAdmin = u.IsAdmin
	}

	if si, ok := subscription.SubscriptionFromContext(r.Context()); ok {
		access.PlanCode = si.PlanCode
	} else if access.IsAuthenticated {
		access.PlanCode = "START"
	}

	return access
}

func parseListParams(r *http.Request) ArticleListParams {
	params := ArticleListParams{
		Page:    1,
		PerPage: 10,
	}

	if page := r.URL.Query().Get("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil && p > 0 {
			params.Page = p
		}
	}

	if perPage := r.URL.Query().Get("perPage"); perPage != "" {
		if pp, err := strconv.Atoi(perPage); err == nil && pp > 0 {
			params.PerPage = pp
		}
	}

	if category := r.URL.Query().Get("category"); category != "" {
		params.CategorySlug = &category
	}

	if tag := r.URL.Query().Get("tag"); tag != "" {
		params.TagSlug = &tag
	}

	if status := r.URL.Query().Get("status"); status != "" {
		params.Status = &status
	}

	if search := r.URL.Query().Get("search"); search != "" {
		params.Search = &search
	}

	return params
}

// ===== Public Endpoints =====

func (c *Controller) GetPublicArticles(w http.ResponseWriter, r *http.Request) {
	params := parseListParams(r)
	viewer := getViewerAccess(r)

	response, err := c.service.GetPublicArticles(context.Background(), params, viewer)
	if err != nil {
		logger.Error("failed to get public articles", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

func (c *Controller) GetPublicArticle(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	viewer := getViewerAccess(r)

	article, err := c.service.GetPublicArticleBySlug(context.Background(), slug, viewer)
	if err != nil {
		if err.Error() == "access denied" {
			http.Error(w, "access denied", http.StatusForbidden)
			return
		}
		logger.Error("failed to get article", "slug", slug, "error", err)
		http.Error(w, "article not found", http.StatusNotFound)
		return
	}

	// Track view asynchronously
	go func() {
		if err := c.service.TrackView(context.Background(), article.Id); err != nil {
			logger.Error("failed to track view", "articleId", article.Id, "error", err)
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(article)
}

func (c *Controller) GetCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := c.service.GetAllCategories(context.Background())
	if err != nil {
		logger.Error("failed to get categories", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(categories)
}

func (c *Controller) GetTags(w http.ResponseWriter, r *http.Request) {
	tags, err := c.service.GetAllTags(context.Background())
	if err != nil {
		logger.Error("failed to get tags", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(tags)
}

func (c *Controller) SubmitFeedback(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid article id", http.StatusBadRequest)
		return
	}

	var req struct {
		Helpful   bool    `json:"helpful"`
		SessionId *string `json:"sessionId,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	feedbackCreate := FeedbackCreate{
		ArticleId: id,
		Helpful:   req.Helpful,
		SessionId: req.SessionId,
	}

	// Check if user is logged in
	if u, ok := auth.UserFromContext(r.Context()); ok {
		feedbackCreate.UserId = &u.Id
	}

	feedback, err := c.service.SubmitFeedback(context.Background(), feedbackCreate)
	if err != nil {
		if err.Error() == "feedback already submitted" {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		logger.Error("failed to submit feedback", "articleId", id, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(feedback)
}

func (c *Controller) GetFeedbackStats(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid article id", http.StatusBadRequest)
		return
	}

	stats, err := c.service.GetFeedbackStats(context.Background(), id)
	if err != nil {
		logger.Error("failed to get feedback stats", "articleId", id, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(stats)
}

// ===== Admin Articles =====

func (c *Controller) PrepareArticle(w http.ResponseWriter, r *http.Request) {
	var req PrepareArticleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req.TitleRu = strings.TrimSpace(req.TitleRu)
	req.DescriptionRu = strings.TrimSpace(req.DescriptionRu)
	req.ContentMarkdown = strings.TrimSpace(req.ContentMarkdown)
	if req.PreviewImageUrl != nil {
		trimmed := strings.TrimSpace(*req.PreviewImageUrl)
		if trimmed == "" {
			req.PreviewImageUrl = nil
		} else {
			req.PreviewImageUrl = &trimmed
		}
	}

	if req.TitleRu == "" {
		http.Error(w, "titleRu is required", http.StatusBadRequest)
		return
	}
	if req.DescriptionRu == "" {
		http.Error(w, "descriptionRu is required", http.StatusBadRequest)
		return
	}
	if req.ContentMarkdown == "" {
		http.Error(w, "contentMarkdownRu is required", http.StatusBadRequest)
		return
	}

	prepared, err := c.service.PrepareArticle(r.Context(), req)
	if err != nil {
		logger.Error("failed to prepare article", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(prepared)
}

func (c *Controller) CreateArticle(w http.ResponseWriter, r *http.Request) {
	var articleCreate ArticleCreate
	if err := json.NewDecoder(r.Body).Decode(&articleCreate); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if articleCreate.Slug == "" || articleCreate.TitleRu == "" || articleCreate.TitleEn == "" {
		http.Error(w, "slug, titleRu, and titleEn are required", http.StatusBadRequest)
		return
	}

	// Set author
	if u, ok := auth.UserFromContext(r.Context()); ok {
		articleCreate.AuthorId = &u.Id
	}

	// Validate status
	if articleCreate.Status == "" {
		articleCreate.Status = "draft"
	}
	if articleCreate.Status != "draft" && articleCreate.Status != "preview" && articleCreate.Status != "authorized" && articleCreate.Status != "paid" && articleCreate.Status != "public" {
		http.Error(w, "invalid status: must be draft, preview, authorized, paid, or public", http.StatusBadRequest)
		return
	}

	article, err := c.service.CreateArticle(context.Background(), articleCreate)
	if err != nil {
		logger.Error("failed to create article", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(article)
}

func (c *Controller) UpdateArticle(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var articleUpdate ArticleUpdate
	if err := json.NewDecoder(r.Body).Decode(&articleUpdate); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	articleUpdate.Id = id

	// Validate status
	if articleUpdate.Status != "" && articleUpdate.Status != "draft" && articleUpdate.Status != "preview" && articleUpdate.Status != "authorized" && articleUpdate.Status != "paid" && articleUpdate.Status != "public" {
		http.Error(w, "invalid status: must be draft, preview, authorized, paid, or public", http.StatusBadRequest)
		return
	}

	article, err := c.service.UpdateArticle(context.Background(), articleUpdate)
	if err != nil {
		logger.Error("failed to update article", "id", id, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(article)
}

func (c *Controller) DeleteArticle(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := c.service.DeleteArticle(context.Background(), id); err != nil {
		logger.Error("failed to delete article", "id", id, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

func (c *Controller) GetAllArticles(w http.ResponseWriter, r *http.Request) {
	params := parseListParams(r)

	response, err := c.service.GetAllArticles(context.Background(), params)
	if err != nil {
		logger.Error("failed to get all articles", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

func (c *Controller) GetArticleById(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	article, err := c.service.GetArticleById(context.Background(), id)
	if err != nil {
		logger.Error("failed to get article by id", "id", id, "error", err)
		http.Error(w, "article not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(article)
}

// ===== Admin Categories =====

func (c *Controller) CreateCategory(w http.ResponseWriter, r *http.Request) {
	var categoryCreate CategoryCreate
	if err := json.NewDecoder(r.Body).Decode(&categoryCreate); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if categoryCreate.Slug == "" || categoryCreate.NameRu == "" || categoryCreate.NameEn == "" {
		http.Error(w, "slug, nameRu, and nameEn are required", http.StatusBadRequest)
		return
	}

	category, err := c.service.CreateCategory(context.Background(), categoryCreate)
	if err != nil {
		logger.Error("failed to create category", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(category)
}

func (c *Controller) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var categoryUpdate CategoryUpdate
	if err := json.NewDecoder(r.Body).Decode(&categoryUpdate); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	categoryUpdate.Id = id

	category, err := c.service.UpdateCategory(context.Background(), categoryUpdate)
	if err != nil {
		logger.Error("failed to update category", "id", id, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(category)
}

func (c *Controller) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := c.service.DeleteCategory(context.Background(), id); err != nil {
		logger.Error("failed to delete category", "id", id, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

// ===== Admin Tags =====

func (c *Controller) CreateTag(w http.ResponseWriter, r *http.Request) {
	var tagCreate TagCreate
	if err := json.NewDecoder(r.Body).Decode(&tagCreate); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if tagCreate.Slug == "" || tagCreate.NameRu == "" || tagCreate.NameEn == "" {
		http.Error(w, "slug, nameRu, and nameEn are required", http.StatusBadRequest)
		return
	}

	tag, err := c.service.CreateTag(context.Background(), tagCreate)
	if err != nil {
		logger.Error("failed to create tag", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(tag)
}

func (c *Controller) UpdateTag(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var tagUpdate TagUpdate
	if err := json.NewDecoder(r.Body).Decode(&tagUpdate); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	tagUpdate.Id = id

	tag, err := c.service.UpdateTag(context.Background(), tagUpdate)
	if err != nil {
		logger.Error("failed to update tag", "id", id, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(tag)
}

func (c *Controller) DeleteTag(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := c.service.DeleteTag(context.Background(), id); err != nil {
		logger.Error("failed to delete tag", "id", id, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

// ===== Image Upload =====

func (c *Controller) UploadImage(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form (max 10MB)
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, "failed to parse form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "image file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Read image data
	imageData, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "failed to read image", http.StatusBadRequest)
		return
	}

	// Upload to MinIO
	imageUrl, err := c.service.UploadImage(context.Background(), imageData, header.Filename)
	if err != nil {
		logger.Error("failed to upload image", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(ImageUploadResponse{Url: imageUrl})
}
