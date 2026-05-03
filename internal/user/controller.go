package user

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/charmbracelet/log"
	"github.com/go-chi/chi/v5"
	"github.com/jourloy/nutri02/pkg/timeutil"
)

var (
	logger = log.NewWithOptions(os.Stderr, log.Options{
		Prefix: "[user]",
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
	router.Route("/user", func(r chi.Router) {
		r.Get("/stats", c.GetStats)
		r.Put("/timezone", c.UpdateTimezone)

		// Email verification routes
		r.Route("/email", func(r chi.Router) {
			r.Post("/request", c.RequestEmailVerification)
			r.Post("/verify", c.VerifyEmail)
			r.Post("/resend", c.ResendVerificationCode)
			r.Get("/status", c.GetEmailStatus)
		})
	})

	logger.Info("╔═════ User")
	logger.Info("║    GET /stats")
	logger.Info("║    PUT /timezone")
	logger.Info("║    POST /email/request")
	logger.Info("║    POST /email/verify")
	logger.Info("║    POST /email/resend")
	logger.Info("║    GET /email/status")
	logger.Info("╚═════")
}

func (c *Controller) GetStats(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	stats, err := c.service.GetUserStats(r.Context(), u.Id)
	if err != nil {
		logger.Error("Error getting user stats", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(stats)
}

// RequestEmailVerification запрашивает код верификации email
func (c *Controller) RequestEmailVerification(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Email string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Email == "" {
		http.Error(w, "email is required", http.StatusBadRequest)
		return
	}

	if err := c.service.RequestEmailVerification(r.Context(), u.Id, req.Email); err != nil {
		logger.Error("Error requesting email verification", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "verification code sent"})
}

// VerifyEmail проверяет код верификации
func (c *Controller) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Code string `json:"code"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Code == "" {
		http.Error(w, "code is required", http.StatusBadRequest)
		return
	}

	updatedUser, err := c.service.VerifyEmail(r.Context(), u.Id, req.Code)
	if err != nil {
		logger.Error("Error verifying email", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(updatedUser)
}

// ResendVerificationCode повторно отправляет код верификации
func (c *Controller) ResendVerificationCode(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := c.service.ResendVerificationCode(r.Context(), u.Id); err != nil {
		logger.Error("Error resending verification code", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "verification code resent"})
}

// GetEmailStatus возвращает статус верификации email
func (c *Controller) GetEmailStatus(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := c.service.GetUser(u.Id)
	if err != nil {
		logger.Error("Error getting user", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	status := map[string]interface{}{
		"email":         user.Email,
		"emailVerified": user.EmailVerified,
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(status)
}

// UpdateTimezone обновляет timezone пользователя
func (c *Controller) UpdateTimezone(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Timezone string `json:"timezone"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Timezone == "" {
		http.Error(w, "timezone is required", http.StatusBadRequest)
		return
	}

	// Validate timezone
	if !timeutil.ValidateTimezone(req.Timezone) {
		http.Error(w, "invalid timezone", http.StatusBadRequest)
		return
	}

	user, err := c.service.UpdateTimezone(r.Context(), u.Id, req.Timezone)
	if err != nil {
		logger.Error("Error updating timezone", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(user)
}
