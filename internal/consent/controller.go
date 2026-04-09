package consent

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/charmbracelet/log"
	"github.com/go-chi/chi/v5"

	"github.com/jourloy/somivyn/internal/auth"
)

var (
	logger = log.NewWithOptions(os.Stderr, log.Options{
		Prefix: "[consent]",
		Level:  log.DebugLevel,
	})
)

type Controller struct {
	service Service
}

func NewController() *Controller {
	return &Controller{service: NewService(NewRepository())}
}

func (c *Controller) RegisterRoutes(router chi.Router) {
	router.Route("/consent", func(r chi.Router) {
		r.Post("/", c.RecordConsent)
		r.Get("/latest", c.GetLatestConsent)
	})

	logger.Info("╔═════ Consent")
	logger.Info("║   POST /  (public)")
	logger.Info("║    GET /latest  (authenticated)")
	logger.Info("╚═════")
}

// extractIP extracts IP address from request headers
func extractIP(r *http.Request) string {
	// Try X-Forwarded-For first (standard proxy header)
	ip := r.Header.Get("X-Forwarded-For")
	if ip != "" {
		return ip
	}

	// Try Cloudflare header
	ip = r.Header.Get("CF-Connecting-IP")
	if ip != "" {
		return ip
	}

	// Fallback to RemoteAddr
	return r.RemoteAddr
}

func (c *Controller) RecordConsent(w http.ResponseWriter, r *http.Request) {
	var req RecordConsentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Extract IP and User-Agent
	ipAddress := extractIP(r)
	userAgent := r.UserAgent()

	// Try to get authenticated user
	var userId *string
	if user, ok := auth.UserFromContext(r.Context()); ok {
		userId = &user.Id
	}

	// Record consent
	record, err := c.service.RecordConsent(
		r.Context(),
		userId,
		ipAddress,
		userAgent,
		req.ConsentGiven,
		req.ConsentType,
	)
	if err != nil {
		logger.Error("Failed to record consent", "error", err)
		http.Error(w, "failed to record consent", http.StatusInternalServerError)
		return
	}

	// Return response
	response := ConsentResponse{
		Success: true,
		Record:  record,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(response)
}

func (c *Controller) GetLatestConsent(w http.ResponseWriter, r *http.Request) {
	// This endpoint requires authentication
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	record, err := c.service.GetLatestConsent(r.Context(), user.Id)
	if err != nil {
		// If no record found, return null
		response := ConsentResponse{
			Success: true,
			Record:  nil,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
		return
	}

	response := ConsentResponse{
		Success: true,
		Record:  record,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}
