package supplement

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/charmbracelet/log"
	"github.com/go-chi/chi/v5"

	"github.com/jourloy/nutri02/internal/auth"
	"github.com/jourloy/nutri02/internal/user"
	"github.com/jourloy/nutri02/pkg/timeutil"
)

var (
	logger = log.NewWithOptions(os.Stderr, log.Options{
		Prefix: "[spmt]",
		Level:  log.DebugLevel,
	})
)

type Controller struct {
	service     Service
	userService user.Service
}

func NewController() *Controller {
	return &Controller{
		service:     NewService(),
		userService: user.NewService(),
	}
}

func (c *Controller) RegisterRoutes(router chi.Router) {
	router.Route("/supplement", func(r chi.Router) {
		// Templates
		r.Get("/templates", c.GetTemplates)

		// Supplements CRUD
		r.Post("/", c.CreateSupplement)
		r.Get("/", c.GetUserSupplements)
		r.Get("/{id}", c.GetSupplement)
		r.Put("/{id}", c.UpdateSupplement)
		r.Delete("/{id}", c.DeleteSupplement)

		// Today's view
		r.Get("/today", c.GetTodayIntakes)

		// Mark as taken
		r.Post("/intake", c.CreateIntake)
		r.Delete("/intake/{id}", c.DeleteIntake)

		// History
		r.Get("/history", c.GetIntakeHistory)

		// Statistics
		r.Get("/stats", c.GetStatistics)
	})

	logger.Info("╔═════ Supplement")
	logger.Info("║    GET /templates")
	logger.Info("║   POST /")
	logger.Info("║    GET /")
	logger.Info("║    GET /{id}")
	logger.Info("║    PUT /{id}")
	logger.Info("║ DELETE /{id}")
	logger.Info("║    GET /today")
	logger.Info("║   POST /intake")
	logger.Info("║ DELETE /intake/{id}")
	logger.Info("║    GET /history?date=YYYY-MM-DD&supplementId=uuid")
	logger.Info("║    GET /stats")
	logger.Info("╚═════")
}

// RegisterInternalRoutes registers internal API routes for Telegram bot
func (c *Controller) RegisterInternalRoutes(router chi.Router) {
	router.Route("/supplement", func(r chi.Router) {
		r.Post("/intake", c.CreateIntakeInternal)
	})

	logger.Info("╔═════ Supplement (Internal)")
	logger.Info("║   POST /intake")
	logger.Info("╚═════")
}

// === Templates ===

func (c *Controller) GetTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := c.service.GetAllTemplates(context.Background())
	if err != nil {
		logger.Error("Error getting templates", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(templates)
}

// === Supplements CRUD ===

func (c *Controller) CreateSupplement(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req SupplementCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	supplement, err := c.service.CreateSupplement(context.Background(), u.Id, req)
	if err != nil {
		logger.Error("Error creating supplement", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(supplement)
}

func (c *Controller) GetUserSupplements(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Query param: active=true|false
	activeOnly := false
	if activeStr := r.URL.Query().Get("active"); activeStr == "true" {
		activeOnly = true
	}

	supplements, err := c.service.GetUserSupplements(context.Background(), u.Id, activeOnly)
	if err != nil {
		logger.Error("Error getting user supplements", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(supplements)
}

func (c *Controller) GetSupplement(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	supplementID := chi.URLParam(r, "id")
	if supplementID == "" {
		http.Error(w, "supplement id is required", http.StatusBadRequest)
		return
	}

	supplement, err := c.service.GetSupplementByID(context.Background(), supplementID, u.Id)
	if err != nil {
		logger.Error("Error getting supplement", "error", err)
		http.Error(w, "supplement not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(supplement)
}

func (c *Controller) UpdateSupplement(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	supplementID := chi.URLParam(r, "id")
	if supplementID == "" {
		http.Error(w, "supplement id is required", http.StatusBadRequest)
		return
	}

	var req SupplementCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	supplement, err := c.service.UpdateSupplement(context.Background(), supplementID, u.Id, req)
	if err != nil {
		logger.Error("Error updating supplement", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(supplement)
}

func (c *Controller) DeleteSupplement(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	supplementID := chi.URLParam(r, "id")
	if supplementID == "" {
		http.Error(w, "supplement id is required", http.StatusBadRequest)
		return
	}

	if err := c.service.DeleteSupplement(context.Background(), supplementID, u.Id); err != nil {
		logger.Error("Error deleting supplement", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// === Today's Intakes ===

func (c *Controller) GetTodayIntakes(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	timezone := timeutil.GetTimezoneOrDefault(u.Timezone)

	intakes, err := c.service.CalculateTodayIntakes(context.Background(), u.Id, timezone)
	if err != nil {
		logger.Error("Error calculating today's intakes", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(intakes)
}

// === Intakes ===

func (c *Controller) CreateIntake(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req IntakeCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Default source to dashboard if not provided
	if req.Source == "" {
		req.Source = "dashboard"
	}

	timezone := timeutil.GetTimezoneOrDefault(u.Timezone)

	intake, err := c.service.CreateIntake(context.Background(), req, u.Id, timezone)
	if err != nil {
		logger.Error("Error creating intake", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(intake)
}

func (c *Controller) CreateIntakeInternal(w http.ResponseWriter, r *http.Request) {
	// This is called by Telegram bot via internal API
	// No user auth, but requires service token (checked by middleware)

	var req struct {
		UserID       string  `json:"userId"`
		SupplementID string  `json:"supplementId"`
		ScheduleID   *string `json:"scheduleId,omitempty"`
		Source       string  `json:"source"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Get user to find timezone
	u, err := c.userService.GetUser(req.UserID)
	if err != nil || u == nil {
		logger.Error("User not found", "userId", req.UserID, "error", err)
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	timezone := timeutil.GetTimezoneOrDefault(u.Timezone)

	intakeReq := IntakeCreateRequest{
		SupplementID: req.SupplementID,
		ScheduleID:   req.ScheduleID,
		TakenAt:      time.Now(),
		Source:       req.Source,
	}

	intake, err := c.service.CreateIntake(context.Background(), intakeReq, req.UserID, timezone)
	if err != nil {
		logger.Error("Error creating intake (internal)", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(intake)
}

func (c *Controller) DeleteIntake(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	intakeID := chi.URLParam(r, "id")
	if intakeID == "" {
		http.Error(w, "intake id is required", http.StatusBadRequest)
		return
	}

	if err := c.service.DeleteIntake(context.Background(), intakeID, u.Id); err != nil {
		logger.Error("Error deleting intake", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (c *Controller) GetIntakeHistory(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Query params
	date := r.URL.Query().Get("date")
	supplementID := r.URL.Query().Get("supplementId")
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	params := IntakeHistoryParams{
		Limit:  100, // Default
		Offset: 0,
	}

	if date != "" {
		params.Date = &date
	}
	if supplementID != "" {
		params.SupplementID = &supplementID
	}
	if limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			params.Limit = limit
		}
	}
	if offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			params.Offset = offset
		}
	}

	intakes, err := c.service.GetIntakeHistory(context.Background(), u.Id, params)
	if err != nil {
		logger.Error("Error getting intake history", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(intakes)
}

// === Statistics ===

func (c *Controller) GetStatistics(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	stats, err := c.service.GetStatistics(context.Background(), u.Id)
	if err != nil {
		logger.Error("Error getting statistics", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(stats)
}
