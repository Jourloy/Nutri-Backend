package feedback

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/go-chi/chi/v5"

	"github.com/jourloy/somivyn/internal/auth"
)

var (
	logger = log.NewWithOptions(os.Stderr, log.Options{
		Prefix: "[fdbk]",
		Level:  log.DebugLevel,
	})
)

type Controller struct {
	service Service
}

func NewController() *Controller {
	return &Controller{
		service: NewService(),
	}
}

func (c *Controller) RegisterRoutes(router chi.Router) {
	router.Route("/feedback", func(r chi.Router) {
		r.Get("/status", c.GetStatus)
		r.Post("/", c.Submit)
		r.Patch("/{id}/viewed", c.UpdateViewed)
	})

	logger.Info("╔═════ Feedback")
	logger.Info("║    GET /status")
	logger.Info("║   POST /")
	logger.Info("║  PATCH /{id}/viewed")
	logger.Info("╚═════")
}

func (c *Controller) GetStatus(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	resp, err := c.service.GetStatus(r.Context(), u.Id)
	if err != nil {
		logger.Error("failed to get feedback status", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

type updateViewedRequest struct {
	Viewed bool `json:"viewed"`
}

func (c *Controller) UpdateViewed(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if !u.IsAdmin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	id := chi.URLParam(r, "id")
	if strings.TrimSpace(id) == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	var req updateViewedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := c.service.SetViewed(r.Context(), id, req.Viewed)
	if err != nil {
		logger.Error("failed to update feedback viewed flag", "error", err, "id", id)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

type submitRequest struct {
	Status  string  `json:"status"`
	Message *string `json:"message"`
}

func (c *Controller) Submit(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req submitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	status := strings.ToLower(strings.TrimSpace(req.Status))
	if status == "" {
		http.Error(w, "status is required", http.StatusBadRequest)
		return
	}

	switch status {
	case "positive", "negative", "dismissed":
	default:
		http.Error(w, "unknown status", http.StatusBadRequest)
		return
	}

	if status == "negative" {
		if req.Message == nil || strings.TrimSpace(*req.Message) == "" {
			http.Error(w, "message is required for negative status", http.StatusBadRequest)
			return
		}
	}

	resp, err := c.service.Submit(r.Context(), u.Id, status, req.Message)
	if err != nil {
		logger.Error("failed to submit feedback", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}
