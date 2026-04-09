package news

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strconv"

	"github.com/charmbracelet/log"
	"github.com/go-chi/chi/v5"

	"github.com/jourloy/somivyn/internal/auth"
)

var (
	logger = log.NewWithOptions(os.Stderr, log.Options{
		Prefix: "[news]",
		Level:  log.DebugLevel,
	})
)

type Controller struct {
	service Service
}

func NewController() (*Controller, error) {
	repo := NewRepository()
	svc := NewService(repo)

	return &Controller{
		service: svc,
	}, nil
}

func (c *Controller) RegisterRoutes(router chi.Router) {
	router.Route("/news", func(r chi.Router) {
		// Public endpoints
		r.Get("/published", c.GetPublishedNews)
		r.Post("/mark-viewed/{id}", c.MarkAsViewed)

		// Admin endpoints
		r.Group(func(r chi.Router) {
			r.Use(requireAdmin)
			r.Post("/", c.CreateNews)
			r.Put("/{id}", c.UpdateNews)
			r.Delete("/{id}", c.DeleteNews)
			r.Get("/", c.GetAllNews)
			r.Get("/{id}", c.GetNewsById)
			r.Get("/preview", c.GetPreviewNews)
			r.Post("/{id}/publish", c.PublishNews)
			r.Post("/{id}/unpublish", c.UnpublishNews)
			r.Post("/{id}/status", c.UpdateNewsStatus)
		})
	})

	logger.Info("╔═════ News")
	logger.Info("║    GET /news/published (get published news with unread count)")
	logger.Info("║   POST /news/mark-viewed/{id} (mark news as viewed)")
	logger.Info("║   POST /news (admin: create news)")
	logger.Info("║    PUT /news/{id} (admin: update news)")
	logger.Info("║ DELETE /news/{id} (admin: delete news)")
	logger.Info("║    GET /news (admin: get all news)")
	logger.Info("║    GET /news/{id} (admin: get news by id)")
	logger.Info("║    GET /news/preview (admin: get preview news)")
	logger.Info("║   POST /news/{id}/publish (admin: publish news)")
	logger.Info("║   POST /news/{id}/unpublish (admin: unpublish news)")
	logger.Info("║   POST /news/{id}/status (admin: update news status)")
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

// ===== Public Endpoints =====

func (c *Controller) GetPublishedNews(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 50 // default
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	response, err := c.service.GetPublishedNews(context.Background(), u.Id, limit)
	if err != nil {
		logger.Error("failed to get published news", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

func (c *Controller) MarkAsViewed(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := c.service.MarkAsViewed(context.Background(), u.Id, id); err != nil {
		logger.Error("failed to mark news as viewed", "newsId", id, "userId", u.Id, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// ===== Admin Endpoints =====

func (c *Controller) CreateNews(w http.ResponseWriter, r *http.Request) {
	var newsCreate NewsCreate
	if err := json.NewDecoder(r.Body).Decode(&newsCreate); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	news, err := c.service.CreateNews(context.Background(), newsCreate)
	if err != nil {
		logger.Error("failed to create news", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(news)
}

func (c *Controller) UpdateNews(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var newsUpdate NewsUpdate
	if err := json.NewDecoder(r.Body).Decode(&newsUpdate); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	newsUpdate.Id = id

	news, err := c.service.UpdateNews(context.Background(), newsUpdate)
	if err != nil {
		logger.Error("failed to update news", "id", id, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(news)
}

func (c *Controller) DeleteNews(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := c.service.DeleteNews(context.Background(), id); err != nil {
		logger.Error("failed to delete news", "id", id, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

func (c *Controller) GetAllNews(w http.ResponseWriter, r *http.Request) {
	includeUnpublished := r.URL.Query().Get("includeUnpublished") == "true"

	newsList, err := c.service.GetAllNews(context.Background(), includeUnpublished)
	if err != nil {
		logger.Error("failed to get all news", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(newsList)
}

func (c *Controller) GetNewsById(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	news, err := c.service.GetNewsById(context.Background(), id)
	if err != nil {
		logger.Error("failed to get news by id", "id", id, "error", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(news)
}

func (c *Controller) PublishNews(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := c.service.PublishNews(context.Background(), id); err != nil {
		logger.Error("failed to publish news", "id", id, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "published"})
}

func (c *Controller) UnpublishNews(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := c.service.UnpublishNews(context.Background(), id); err != nil {
		logger.Error("failed to unpublish news", "id", id, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "unpublished"})
}

func (c *Controller) GetPreviewNews(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 50 // default
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	newsList, err := c.service.GetPreviewNews(context.Background(), limit)
	if err != nil {
		logger.Error("failed to get preview news", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(newsList)
}

func (c *Controller) UpdateNewsStatus(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate status
	if req.Status != "draft" && req.Status != "preview" && req.Status != "published" {
		http.Error(w, "invalid status: must be draft, preview, or published", http.StatusBadRequest)
		return
	}

	if err := c.service.UpdateNewsStatus(context.Background(), id, req.Status); err != nil {
		logger.Error("failed to update news status", "id", id, "status", req.Status, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": req.Status})
}
