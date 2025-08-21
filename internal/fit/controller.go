package fit

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
		Prefix: "[ fit]",
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
	router.Route("/fit", func(r chi.Router) {
		r.Post("/", c.Create)
		r.Put("/", c.Update)
		r.Get("/", c.Get)
	})

	logger.Info("╔═════ Fit")
	logger.Info("║   POST /")
	logger.Info("║    PUT /")
	logger.Info("║    GET /")
	logger.Info("╚═════")
}

func (c *Controller) Create(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var fc FitProfileCreate
	if err := json.NewDecoder(r.Body).Decode(&fc); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fc.UserId = u.Id

	resp, err := c.service.CreateFitProfile(fc)
	if err != nil {
		logger.Error("Error creating fit profile", "error", err)
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

	var fu FitProfileCreate
	if err := json.NewDecoder(r.Body).Decode(&fu); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fu.UserId = u.Id

	resp, err := c.service.UpdateFitProfile(context.Background(), fu, u.Id)
	if err != nil {
		logger.Error("Error updating fit profile", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

func (c *Controller) Get(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	resp, err := c.service.GetFitProfileByUser(u.Id)
	if err != nil {
		logger.Error("Error login", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
