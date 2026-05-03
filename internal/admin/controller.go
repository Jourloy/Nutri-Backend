package admin

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/jourloy/nutri02/internal/auth"
)

var (
	logger = log.NewWithOptions(os.Stderr, log.Options{
		Prefix: "[admn]",
		Level:  log.DebugLevel,
	})
)

type Controller struct {
	service Service
}

func NewController() *Controller {
	service := NewService()
	return &Controller{service: service}
}

func (c *Controller) RegisterRoutes(router chi.Router) {
	router.Route("/admin", func(r chi.Router) {
		// Dashboard
		r.Get("/dashboard", c.GetDashboard)

		// Users
		r.Get("/users", c.GetUsers)
		r.Post("/users", c.CreateUserWithSubscription)
		r.Get("/users/{userId}", c.GetUserDetails)
		r.Post("/users/{userId}/subscription/grant", c.GrantUserSubscription)
		r.Delete("/users/{userId}", c.DeleteUser)

		// Plans
		r.Patch("/plans/{planId}/price", c.UpdatePlanPrice)
		r.Patch("/plans/{planId}/features", c.UpdatePlanFeatures)
		r.Patch("/users/{userId}/subscription/price", c.UpdateUserSubscriptionPrice)

		// Notifications
		r.Get("/notifications", c.GetNotifications)
		r.Post("/notifications", c.CreateNotification)
		r.Post("/notifications/{id}/send", c.SendNotification)
	})

	logger.Info("╔═════ Admin")
	logger.Info("║ GET    /admin/dashboard")
	logger.Info("║ GET    /admin/users")
	logger.Info("║ POST   /admin/users")
	logger.Info("║ GET    /admin/users/{userId}")
	logger.Info("║ POST   /admin/users/{userId}/subscription/grant")
	logger.Info("║ DELETE /admin/users/{userId}")
	logger.Info("║ PATCH  /admin/plans/{planId}/price")
	logger.Info("║ PATCH  /admin/plans/{planId}/features")
	logger.Info("║ PATCH  /admin/users/{userId}/subscription/price")
	logger.Info("║ GET    /admin/notifications")
	logger.Info("║ POST   /admin/notifications")
	logger.Info("║ POST   /admin/notifications/{id}/send")
	logger.Info("╚═════")
}

func parseUserSortBy(raw string) UserSortBy {
	switch UserSortBy(strings.ToLower(strings.TrimSpace(raw))) {
	case UserSortByID:
		return UserSortByID
	case UserSortByUsername:
		return UserSortByUsername
	case UserSortByEmail:
		return UserSortByEmail
	case UserSortByLocale:
		return UserSortByLocale
	case UserSortByPlanName:
		return UserSortByPlanName
	case UserSortBySubStatus:
		return UserSortBySubStatus
	case UserSortBySubPeriodEnd:
		return UserSortBySubPeriodEnd
	case UserSortByLoginedAt:
		return UserSortByLoginedAt
	default:
		return UserSortByCreatedAt
	}
}

func parseSortOrder(raw string) SortOrder {
	if SortOrder(strings.ToLower(strings.TrimSpace(raw))) == SortOrderAsc {
		return SortOrderAsc
	}
	return SortOrderDesc
}

// GetDashboard возвращает статистику для дашборда
func (c *Controller) GetDashboard(w http.ResponseWriter, r *http.Request) {
	stats, err := c.service.GetDashboardStats(r.Context())
	if err != nil {
		logger.Error("Error getting dashboard stats", "error", err)
		http.Error(w, `{"error": "internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// GetUsers возвращает список пользователей с пагинацией
func (c *Controller) GetUsers(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	sortBy := parseUserSortBy(r.URL.Query().Get("sort_by"))
	sortOrder := parseSortOrder(r.URL.Query().Get("sort_order"))

	users, total, err := c.service.GetAllUsers(r.Context(), limit, offset, sortBy, sortOrder)
	if err != nil {
		logger.Error("Error getting users", "error", err)
		http.Error(w, `{"error": "internal server error"}`, http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"users":      users,
		"total":      total,
		"limit":      limit,
		"offset":     offset,
		"sort_by":    sortBy,
		"sort_order": sortOrder,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (c *Controller) GetUserDetails(w http.ResponseWriter, r *http.Request) {
	userId := chi.URLParam(r, "userId")
	if userId == "" {
		http.Error(w, `{"error": "user id is required"}`, http.StatusBadRequest)
		return
	}

	details, err := c.service.GetUserDetails(r.Context(), userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, `{"error": "user not found"}`, http.StatusNotFound)
			return
		}
		logger.Error("Error getting user details", "error", err)
		http.Error(w, `{"error": "internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(details)
}

func (c *Controller) GrantUserSubscription(w http.ResponseWriter, r *http.Request) {
	userId := chi.URLParam(r, "userId")
	if userId == "" {
		http.Error(w, `{"error": "user id is required"}`, http.StatusBadRequest)
		return
	}

	var req GrantUserSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}
	if req.PlanId <= 0 || req.DurationDays <= 0 {
		http.Error(w, `{"error": "plan_id and duration_days must be greater than zero"}`, http.StatusBadRequest)
		return
	}

	subscription, err := c.service.GrantUserSubscription(r.Context(), userId, req.PlanId, req.DurationDays)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, `{"error": "user or plan not found"}`, http.StatusNotFound)
			return
		}
		logger.Error("Error granting subscription", "error", err)
		http.Error(w, `{"error": "failed to grant subscription"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":      true,
		"subscription": subscription,
	})
}

func (c *Controller) DeleteUser(w http.ResponseWriter, r *http.Request) {
	userId := chi.URLParam(r, "userId")
	if userId == "" {
		http.Error(w, `{"error": "user id is required"}`, http.StatusBadRequest)
		return
	}

	if err := c.service.DeleteUser(r.Context(), userId); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, `{"error": "user not found"}`, http.StatusNotFound)
			return
		}
		logger.Error("Error deleting user", "error", err)
		http.Error(w, `{"error": "failed to delete user"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// CreateUserWithSubscription создает пользователя с подпиской
func (c *Controller) CreateUserWithSubscription(w http.ResponseWriter, r *http.Request) {
	var req UserWithSubscription
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Валидация
	if req.Username == "" || req.Password == "" || req.PlanId == 0 || req.DurationMs == 0 {
		http.Error(w, `{"error": "missing required fields"}`, http.StatusBadRequest)
		return
	}

	// Хешируем пароль используя bcrypt (как в auth модуле)
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		logger.Error("Error hashing password", "error", err)
		http.Error(w, `{"error": "internal server error"}`, http.StatusInternalServerError)
		return
	}

	if err := c.service.CreateUserWithSubscription(
		r.Context(),
		req.Username,
		string(passwordHash),
		req.Email,
		req.PlanId,
		req.DurationMs,
	); err != nil {
		logger.Error("Error creating user with subscription", "error", err)
		http.Error(w, `{"error": "failed to create user"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"success": true}`))
}

// UpdatePlanPrice обновляет цену тарифа
func (c *Controller) UpdatePlanPrice(w http.ResponseWriter, r *http.Request) {
	planIdStr := chi.URLParam(r, "planId")
	planId, err := strconv.ParseInt(planIdStr, 10, 64)
	if err != nil {
		http.Error(w, `{"error": "invalid plan id"}`, http.StatusBadRequest)
		return
	}

	var req UpdatePlanPrice
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	if err := c.service.UpdatePlanPrice(r.Context(), planId, req.AmountMinor); err != nil {
		logger.Error("Error updating plan price", "error", err)
		http.Error(w, `{"error": "failed to update price"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"success": true}`))
}

// UpdatePlanFeatures обновляет возможности тарифа
func (c *Controller) UpdatePlanFeatures(w http.ResponseWriter, r *http.Request) {
	planIdStr := chi.URLParam(r, "planId")
	planId, err := strconv.ParseInt(planIdStr, 10, 64)
	if err != nil {
		http.Error(w, `{"error": "invalid plan id"}`, http.StatusBadRequest)
		return
	}

	var features map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&features); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	if err := c.service.UpdatePlanFeatures(r.Context(), planId, features); err != nil {
		logger.Error("Error updating plan features", "error", err)
		http.Error(w, `{"error": "failed to update features"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"success": true}`))
}

// UpdateUserSubscriptionPrice обновляет цену подписки для конкретного пользователя
func (c *Controller) UpdateUserSubscriptionPrice(w http.ResponseWriter, r *http.Request) {
	userId := chi.URLParam(r, "userId")

	var req UpdatePlanPrice
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	if err := c.service.UpdateUserSubscriptionPrice(r.Context(), userId, req.AmountMinor); err != nil {
		logger.Error("Error updating user subscription price", "error", err)
		http.Error(w, `{"error": "failed to update price"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"success": true}`))
}

// CreateNotification создает новое уведомление
func (c *Controller) CreateNotification(w http.ResponseWriter, r *http.Request) {
	// Получаем ID администратора из контекста
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, `{"error": "unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req AdminNotificationCreate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Валидация
	if req.Title == "" || req.Message == "" || req.TargetAudience == "" {
		http.Error(w, `{"error": "missing required fields"}`, http.StatusBadRequest)
		return
	}

	notification, err := c.service.CreateNotification(r.Context(), user.Id, &req)
	if err != nil {
		logger.Error("Error creating notification", "error", err)
		http.Error(w, `{"error": "failed to create notification"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(notification)
}

// GetNotifications возвращает список уведомлений
func (c *Controller) GetNotifications(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	notifications, err := c.service.GetNotifications(r.Context(), limit, offset)
	if err != nil {
		logger.Error("Error getting notifications", "error", err)
		http.Error(w, `{"error": "internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notifications)
}

// SendNotification отправляет уведомление
func (c *Controller) SendNotification(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, `{"error": "invalid notification id"}`, http.StatusBadRequest)
		return
	}

	if err := c.service.SendNotification(r.Context(), id); err != nil {
		logger.Error("Error sending notification", "error", err)
		http.Error(w, `{"error": "failed to send notification"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"success": true}`))
}
