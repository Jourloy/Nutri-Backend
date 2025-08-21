package product

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
		Prefix: "[prct]",
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
	router.Route("/product", func(r chi.Router) {
		r.Post("/", c.Create)
		r.Get("/all", c.GetAll)
		r.Get("/today", c.GetAllByToday)
		r.Get("/search", c.Search)
	})

	logger.Info("╔═════ Product")
	logger.Info("║   POST /")
	logger.Info("║    GET /all")
	logger.Info("║    GET /today")
	logger.Info("║    GET /search?name=")
	logger.Info("╚═════")
}

func (c *Controller) Create(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var pc ProductCreate
	if err := json.NewDecoder(r.Body).Decode(&pc); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	pc.UserId = u.Id

	resp, err := c.service.CreateProduct(context.Background(), pc)
	if err != nil {
		logger.Error("Error creating product", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

func (c *Controller) GetAll(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	resp, err := c.service.GetAll(context.Background(), u.Id)
	if err != nil {
		logger.Error("Error get all", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (c *Controller) GetAllByToday(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	resp, err := c.service.GetAllByToday(context.Background(), u.Id)
	if err != nil {
		logger.Error("Error get all by today", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (c *Controller) Search(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var name *string
	if r.URL.Query().Get("name") != "" {
		v := r.URL.Query().Get("name")
		name = &v
	}

	resp, err := c.service.GetLikeName(context.Background(), *name, u.Id)
	if err != nil {
		logger.Error("Error search by name", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
