package subscription

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
		Prefix: "[subs]",
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
    router.Route("/subscription", func(r chi.Router) {
        r.Post("/", c.Create)
        r.Put("/", c.Update)
        r.Delete("/{id}", c.Delete)
        r.Get("/", c.GetByUser)
    })

    logger.Info("╔═════ Subscription")
    logger.Info("║   POST /")
    logger.Info("║    PUT /")
    logger.Info("║ DELETE /{id}")
    logger.Info("║    GET /")
    logger.Info("╚═════")
}

func (c *Controller) Create(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var sc SubscriptionCreate
	if err := json.NewDecoder(r.Body).Decode(&sc); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	sc.UserId = u.Id

	resp, err := c.service.Create(context.Background(), sc)
	if err != nil {
		logger.Error("Error creating subscription", "error", err)
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

	var s Subscription
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.UserId = u.Id

	resp, err := c.service.Update(context.Background(), s)
	if err != nil {
		logger.Error("Error updating subscription", "error", err)
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

	idParam := chi.URLParam(r, "id")
	if idParam == "" {
		http.Error(w, "not found subscription id", http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := c.service.Delete(context.Background(), id, u.Id); err != nil {
		logger.Error("Error deleting subscription", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

    w.WriteHeader(http.StatusOK)
}

func (c *Controller) GetByUser(w http.ResponseWriter, r *http.Request) {
    u, ok := auth.UserFromContext(r.Context())
    if !ok {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }
    userID := u.Id
    if u.IsAdmin {
        if q := r.URL.Query().Get("userId"); q != "" {
            userID = q
        }
    }
    resp, err := c.service.GetByUser(context.Background(), userID)
    if err != nil {
        logger.Error("Error get subscription by user", "error", err)
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    w.WriteHeader(http.StatusOK)
    _ = json.NewEncoder(w).Encode(resp)
}
