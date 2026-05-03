package internal_api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/charmbracelet/log"
	"github.com/go-chi/chi/v5"

	"github.com/jourloy/nutri02/internal/ai"
	"github.com/jourloy/nutri02/internal/product"
	"github.com/jourloy/nutri02/internal/telegram"
	"github.com/jourloy/nutri02/internal/user"
	"github.com/jourloy/nutri02/pkg/timeutil"
)

var (
	logger = log.NewWithOptions(os.Stderr, log.Options{
		Prefix: "[intl]",
		Level:  log.DebugLevel,
	})
)

type Controller struct {
	aiService       ai.Service
	productService  product.Service
	telegramService telegram.Service
	userService     user.Service
}

func NewController() (*Controller, error) {
	aiRepo := ai.NewRepository()
	aiSvc, err := ai.NewServiceFromConfig(aiRepo)
	if err != nil {
		return nil, err
	}

	return &Controller{
		aiService:       aiSvc,
		productService:  product.NewService(),
		telegramService: telegram.NewService(),
		userService:     user.NewService(),
	}, nil
}

func (c *Controller) RegisterRoutes(router chi.Router) {
	router.Route("/internal", func(r chi.Router) {
		r.Post("/ai/analyze-food", c.AnalyzeFoodImage)
		r.Post("/product", c.CreateProduct)
		r.Get("/user-by-telegram-id", c.GetUserByTelegramId)
		r.Get("/products-today-count", c.GetProductsTodayCount)
	})

	logger.Info("╔═════ Internal API")
	logger.Info("║   POST /ai/analyze-food")
	logger.Info("║   POST /product")
	logger.Info("║    GET /user-by-telegram-id")
	logger.Info("║    GET /products-today-count")
	logger.Info("╚═════")
}

// AnalyzeFoodImage handles food image analysis for internal services (Telegram bot)
func (c *Controller) AnalyzeFoodImage(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form (max 10MB)
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, "failed to parse form", http.StatusBadRequest)
		return
	}

	// Get userId from form
	userId := r.FormValue("userId")
	if userId == "" {
		http.Error(w, "userId is required", http.StatusBadRequest)
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

	// Get total weight (optional)
	var totalWeight *float64
	totalWeightStr := r.FormValue("totalWeight")
	if totalWeightStr != "" {
		weight, err := strconv.ParseFloat(totalWeightStr, 64)
		if err == nil && weight > 0 {
			totalWeight = &weight
		}
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

	// Get language (optional)
	language := r.FormValue("language")
	if language == "" {
		language = "ru"
	}

	// Call service
	result, err := c.aiService.AnalyzeFoodImage(context.Background(), userId, imageData, totalWeight, userPrompt, language)
	if err != nil {
		logger.Error("food analysis failed", "userId", userId, "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(result)
}

// InternalProductCreate represents the request body for creating a product via internal API
type InternalProductCreate struct {
	UserId           string   `json:"userId"`
	Name             string   `json:"name"`
	Amount           int64    `json:"amount"`
	Unit             string   `json:"unit"`
	Calories         float64  `json:"calories"`
	Protein          float64  `json:"protein"`
	Fat              float64  `json:"fat"`
	Carbs            float64  `json:"carbs"`
	Fiber            *float64 `json:"fiber"`
	Cholesterol      *float64 `json:"cholesterol"`
	BasicCalories    float64  `json:"basicCalories"`
	BasicProtein     float64  `json:"basicProtein"`
	BasicFat         float64  `json:"basicFat"`
	BasicCarbs       float64  `json:"basicCarbs"`
	BasicFiber       *float64 `json:"basicFiber"`
	BasicCholesterol *float64 `json:"basicCholesterol"`
	IsWater          bool     `json:"isWater"`
	MealType         *string  `json:"mealType,omitempty"`
}

// CreateProduct handles product creation for internal services
func (c *Controller) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var req InternalProductCreate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.UserId == "" {
		http.Error(w, "userId is required", http.StatusBadRequest)
		return
	}

	// Get user to find timezone
	u, err := c.userService.GetUser(req.UserId)
	if err != nil || u == nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	// Get today's date based on user's timezone
	today := timeutil.CurrentDateForTimezone(timeutil.GetTimezoneOrDefault(u.Timezone))

	// Create product
	pc := product.ProductCreate{
		Name:             req.Name,
		Amount:           req.Amount,
		Unit:             req.Unit,
		Calories:         req.Calories,
		Protein:          req.Protein,
		Fat:              req.Fat,
		Carbs:            req.Carbs,
		Fiber:            req.Fiber,
		Cholesterol:      req.Cholesterol,
		BasicCalories:    req.BasicCalories,
		BasicProtein:     req.BasicProtein,
		BasicFat:         req.BasicFat,
		BasicCarbs:       req.BasicCarbs,
		BasicFiber:       req.BasicFiber,
		BasicCholesterol: req.BasicCholesterol,
		IsWater:          req.IsWater,
		MealType:         req.MealType,
		UserId:           req.UserId,
	}

	result, err := c.productService.CreateProduct(context.Background(), pc, today)
	if err != nil {
		logger.Error("failed to create product", "userId", req.UserId, "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(result)
}

// GetUserByTelegramId returns user info by telegram ID
func (c *Controller) GetUserByTelegramId(w http.ResponseWriter, r *http.Request) {
	telegramId := r.URL.Query().Get("telegramId")
	if telegramId == "" {
		http.Error(w, "telegramId is required", http.StatusBadRequest)
		return
	}

	// Get telegram profile by telegram_id
	profile, err := c.telegramService.GetByTelegramId(context.Background(), telegramId)
	if err != nil || profile == nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	// Get user
	u, err := c.userService.GetUser(profile.UserId)
	if err != nil || u == nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	// Return limited user info
	response := map[string]interface{}{
		"userId":   u.Id,
		"username": u.Username,
		"locale":   u.Locale,
		"timezone": u.Timezone,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

// GetProductsTodayCount returns the count of products logged today for a user
func (c *Controller) GetProductsTodayCount(w http.ResponseWriter, r *http.Request) {
	userId := r.URL.Query().Get("userId")
	if userId == "" {
		http.Error(w, "userId is required", http.StatusBadRequest)
		return
	}

	date := r.URL.Query().Get("date")
	if date == "" {
		http.Error(w, "date is required (YYYY-MM-DD)", http.StatusBadRequest)
		return
	}

	products, err := c.productService.GetAllByDate(context.Background(), date, userId)
	if err != nil {
		logger.Error("failed to get products", "userId", userId, "date", date, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"count":       len(products),
		"hasProducts": len(products) > 0,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}
