package analytics

import (
    "encoding/json"
    "net/http"
    "os"
    "strconv"
    "time"

    "github.com/charmbracelet/log"
    "github.com/go-chi/chi/v5"

    "github.com/jourloy/nutri-backend/internal/auth"
)

var (
    logger = log.NewWithOptions(os.Stderr, log.Options{ Prefix: "[anal]", Level: log.DebugLevel })
)

type Controller struct { service Service }

func NewController() *Controller { return &Controller{service: NewService()} }

func (c *Controller) RegisterRoutes(router chi.Router) {
    router.Route("/analytics", func(r chi.Router) {
        r.Get("/series", c.GetSeries)
    })
    logger.Info("╔═════ Analytics")
    logger.Info("║    GET /series?end=&days=")
    logger.Info("╚═════")
}

func (c *Controller) GetSeries(w http.ResponseWriter, r *http.Request) {
    u, ok := auth.UserFromContext(r.Context())
    if !ok { http.Error(w, "unauthorized", http.StatusUnauthorized); return }
    endStr := r.URL.Query().Get("end")
    daysStr := r.URL.Query().Get("days")
    end := time.Now()
    if endStr != "" { if t, err := time.Parse("2006-01-02", endStr); err == nil { end = t } }
    days := 7
    if daysStr != "" { if v, err := strconv.Atoi(daysStr); err == nil { days = v } }
    res, err := c.service.GetSeries(r.Context(), u.Id, end, days)
    if err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
    w.WriteHeader(http.StatusOK)
    _ = json.NewEncoder(w).Encode(res)
}
