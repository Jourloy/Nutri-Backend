package template

import (
	"context"
	"encoding/json"
	"net/http"
	"os"

	"github.com/charmbracelet/log"
	"github.com/go-chi/chi/v5"
	"github.com/jourloy/nutri-backend/internal/auth"
)

var (
	logger = log.NewWithOptions(os.Stderr, log.Options{
		Prefix: "[tmpl]",
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
	router.Route("/template", func(r chi.Router) {
		r.Get("/search", c.Search)
	})

	logger.Info("╔═════ Template")
	logger.Info("║    GET /search?name=")
	logger.Info("╚═════")
}

func (c *Controller) Search(w http.ResponseWriter, r *http.Request) {
	_, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var name *string
	if r.URL.Query().Get("name") != "" {
		v := r.URL.Query().Get("name")
		name = &v
	}

	resp, err := c.service.GetLikeName(context.Background(), *name)
	if err != nil {
		logger.Error("Error search by name", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
