package orders

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
		Prefix: "[orders]",
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
	router.Route("/orders", func(r chi.Router) {
		r.Post("/create", c.Create)
	})

	logger.Info("╔═════ Orders")
	logger.Info("║   POST /create")
	logger.Info("╚═════")
}

func (c *Controller) Create(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req OrderCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := c.service.Create(context.Background(), u.Id, req)
	if err != nil {
		logger.Error("Error creating order", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
