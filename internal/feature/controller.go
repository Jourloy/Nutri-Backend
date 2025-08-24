package feature

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
		Prefix: "[feat]",
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
	router.Route("/feature", func(r chi.Router) {
		r.Post("/", c.Create)
		r.Put("/", c.Update)
		r.Delete("/{key}", c.Delete)
	})

	logger.Info("╔═════ Feature")
	logger.Info("║   POST /")
	logger.Info("║    PUT /")
	logger.Info("║ DELETE /{key}")
	logger.Info("╚═════")
}

func (c *Controller) Create(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if !u.IsAdmin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	var f Feature
	if err := json.NewDecoder(r.Body).Decode(&f); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := c.service.Create(context.Background(), f)
	if err != nil {
		logger.Error("Error creating feature", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

func (c *Controller) Update(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if !u.IsAdmin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	var f Feature
	if err := json.NewDecoder(r.Body).Decode(&f); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := c.service.Update(context.Background(), f)
	if err != nil {
		logger.Error("Error updating feature", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (c *Controller) Delete(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if !u.IsAdmin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	key := chi.URLParam(r, "key")
	if key == "" {
		http.Error(w, "not found feature key", http.StatusBadRequest)
		return
	}

	if err := c.service.Delete(context.Background(), key); err != nil {
		logger.Error("Error deleting feature", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}
