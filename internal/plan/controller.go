package plan

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strconv"

	"github.com/charmbracelet/log"
	"github.com/go-chi/chi/v5"

	"github.com/jourloy/nutri-backend/internal/auth"
)

var (
	logger = log.NewWithOptions(os.Stderr, log.Options{
		Prefix: "[plan]",
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
	router.Route("/plan", func(r chi.Router) {
		r.Post("/", c.Create)
		r.Put("/", c.Update)
		r.Delete("/{id}", c.Delete)
		r.Get("/all", c.GetAll)
	})

	logger.Info("╔═════ Plan")
	logger.Info("║   POST /")
	logger.Info("║    PUT /")
	logger.Info("║ DELETE /{id}")
	logger.Info("║    GET /all")
	logger.Info("╚═════")
}

func (c *Controller) GetAll(w http.ResponseWriter, r *http.Request) {
	_, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	resp, err := c.service.GetAllActive(context.Background())
	if err != nil {
		logger.Error("Error get all plans", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
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

	var pc PlanCreate
	if err := json.NewDecoder(r.Body).Decode(&pc); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := c.service.Create(context.Background(), pc)
	if err != nil {
		logger.Error("Error creating plan", "error", err)
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

	var p Plan
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := c.service.Update(context.Background(), p)
	if err != nil {
		logger.Error("Error updating plan", "error", err)
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

	idParam := chi.URLParam(r, "id")
	if idParam == "" {
		http.Error(w, "not found plan id", http.StatusBadRequest)
		return
	}

	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := c.service.Delete(context.Background(), id); err != nil {
		logger.Error("Error deleting plan", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}
