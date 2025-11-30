package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/charmbracelet/log"
	"github.com/go-chi/chi/v5"

	"github.com/jourloy/nutri-backend/internal/auth"
)

var (
	logger = log.NewWithOptions(os.Stderr, log.Options{
		Prefix: "[ai]",
		Level:  log.DebugLevel,
	})
)

type Controller struct {
	service Service
}

func NewController() (*Controller, error) {
	repo := NewRepository()
	svc, err := NewService(repo)
	if err != nil {
		return nil, fmt.Errorf("failed to create AI service: %w", err)
	}

	return &Controller{
		service: svc,
	}, nil
}

func (c *Controller) RegisterRoutes(router chi.Router) {
	router.Route("/ai", func(r chi.Router) {
		// Food analysis
		r.Post("/analyze-food", c.AnalyzeFoodImage)
		r.Post("/analyze-food-text", c.AnalyzeFoodByText)
		r.Get("/analysis-history", c.GetAnalysisHistory)
		r.Get("/analysis/{id}", c.GetAnalysisById)

		// Limits
		r.Get("/limit-status", c.GetLimitStatus)
	})

	logger.Info("╔═════ AI")
	logger.Info("║   POST /analyze-food (multipart: image, totalWeight, userPrompt)")
	logger.Info("║   POST /analyze-food-text (json: foodName, foodDescription, totalWeight, language)")
	logger.Info("║    GET /analysis-history?limit=10")
	logger.Info("║    GET /analysis/{id}")
	logger.Info("║    GET /limit-status?type=food_analysis")
	logger.Info("╚═════")
}

// AnalyzeFoodImage handles food image analysis requests
func (c *Controller) AnalyzeFoodImage(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse multipart form (max 10MB)
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, "failed to parse form", http.StatusBadRequest)
		return
	}

	// Get image file
	file, _, err := r.FormFile("image")
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

	// Get total weight (optional - AI will estimate if not provided)
	var totalWeight *float64
	totalWeightStr := r.FormValue("totalWeight")
	if totalWeightStr != "" {
		weight, err := strconv.ParseFloat(totalWeightStr, 64)
		if err != nil {
			http.Error(w, "invalid totalWeight", http.StatusBadRequest)
			return
		}
		if weight <= 0 {
			http.Error(w, "totalWeight must be greater than 0", http.StatusBadRequest)
			return
		}
		totalWeight = &weight
	}

	// Get user prompt (optional)
	userPrompt := r.FormValue("userPrompt")
	if userPrompt == "" {
		if totalWeight == nil {
			userPrompt = "Please analyze this food image, identify the food, estimate the portion size/weight, and provide nutritional information"
		} else {
			userPrompt = "Please analyze this food image and provide nutritional information"
		}
	}

	// Get language (optional, defaults to "en")
	language := r.FormValue("language")
	if language == "" {
		language = "en"
	}

	// Call service
	result, err := c.service.AnalyzeFoodImage(context.Background(), u.Id, imageData, totalWeight, userPrompt, language)
	if err != nil {
		logger.Error("food analysis failed", "userId", u.Id, "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(result)
}

// AnalyzeFoodByText handles food text analysis requests
func (c *Controller) AnalyzeFoodByText(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse JSON body
	var req struct {
		FoodName        string  `json:"foodName"`
		FoodDescription string  `json:"foodDescription"`
		TotalWeight     float64 `json:"totalWeight"`
		Language        string  `json:"language"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.FoodName == "" {
		http.Error(w, "foodName is required", http.StatusBadRequest)
		return
	}

	if req.TotalWeight <= 0 {
		http.Error(w, "totalWeight must be greater than 0", http.StatusBadRequest)
		return
	}

	// Default language
	if req.Language == "" {
		req.Language = "en"
	}

	// Call service
	result, err := c.service.AnalyzeFoodByText(
		context.Background(),
		u.Id,
		req.FoodName,
		req.FoodDescription,
		req.TotalWeight,
		req.Language,
	)
	if err != nil {
		logger.Error("food text analysis failed", "userId", u.Id, "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(result)
}

// GetAnalysisHistory returns user's analysis history
func (c *Controller) GetAnalysisHistory(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse limit parameter
	limitStr := r.URL.Query().Get("limit")
	limit := 10 // default
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	history, err := c.service.GetUserAnalysisHistory(context.Background(), u.Id, limit)
	if err != nil {
		logger.Error("failed to get analysis history", "userId", u.Id, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(history)
}

// GetAnalysisById returns a specific analysis log
func (c *Controller) GetAnalysisById(w http.ResponseWriter, r *http.Request) {
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

	repo := NewRepository()
	log, err := repo.GetAnalysisLogById(context.Background(), id)
	if err != nil {
		http.Error(w, "analysis log not found", http.StatusNotFound)
		return
	}

	// Verify ownership
	if log.UserId != u.Id {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(log)
}

// GetLimitStatus returns user's current limit status
func (c *Controller) GetLimitStatus(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	requestType := r.URL.Query().Get("type")
	if requestType == "" {
		requestType = "food_analysis" // default
	}

	limitResult, err := c.service.CheckUserLimit(context.Background(), u.Id, requestType)
	if err != nil {
		logger.Error("failed to check limit", "userId", u.Id, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(limitResult)
}
