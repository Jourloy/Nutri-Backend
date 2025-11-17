package user

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/charmbracelet/log"
	"github.com/go-chi/chi/v5"
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
	})

	logger.Info("╔═════ User")
	logger.Info("║    GET /stats")
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
