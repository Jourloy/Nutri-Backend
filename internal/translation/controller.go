package translation

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/go-chi/chi/v5"

	"github.com/jourloy/nutri02/internal/auth"
)

var (
	logger = log.NewWithOptions(os.Stderr, log.Options{
		Prefix: "[trns]",
		Level:  log.DebugLevel,
	})
)

type Controller struct {
	service Service
}

func NewController() *Controller {
	return &Controller{service: NewService()}
}

func (c *Controller) RegisterRoutes(router chi.Router) {
	router.Route("/translation", func(r chi.Router) {
		r.Get("/{locale}", c.GetByLocale)
		r.Post("/", c.Upsert)
		r.Delete("/", c.Delete)
	})

	logger.Info("╔═════ Translation")
	logger.Info("║    GET /{locale}")
	logger.Info("║   POST /")
	logger.Info("║ DELETE /")
	logger.Info("╚═════")
}

func (c *Controller) GetByLocale(w http.ResponseWriter, r *http.Request) {
	locale := chi.URLParam(r, "locale")
	if locale == "" {
		locale = "ru"
	}
	payload, err := c.service.GetByLocale(r.Context(), locale)
	if err != nil {
		logger.Error("failed to load translations", "locale", locale, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]any{
		"locale":     strings.ToLower(locale),
		"namespaces": payload.Namespaces,
	}
	if !payload.UpdatedAt.IsZero() {
		response["updatedAt"] = payload.UpdatedAt
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

func (c *Controller) Upsert(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok || !u.IsAdmin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	var req UpsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Namespace) == "" || strings.TrimSpace(req.Key) == "" || strings.TrimSpace(req.Locale) == "" {
		http.Error(w, "namespace, key and locale are required", http.StatusBadRequest)
		return
	}

	item, err := c.service.Upsert(r.Context(), req)
	if err != nil {
		logger.Error("failed to upsert translation", "req", req, "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(item)
}

func (c *Controller) Delete(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok || !u.IsAdmin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	var req DeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Namespace) == "" || strings.TrimSpace(req.Key) == "" || strings.TrimSpace(req.Locale) == "" {
		http.Error(w, "namespace, key and locale are required", http.StatusBadRequest)
		return
	}

	if err := c.service.Delete(r.Context(), req); err != nil {
		logger.Error("failed to delete translation", "req", req, "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
