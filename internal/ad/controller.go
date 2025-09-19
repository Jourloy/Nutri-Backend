package ad

import (
    "encoding/json"
    "net/http"
    "os"

    "github.com/charmbracelet/log"
    "github.com/go-chi/chi/v5"
)

var (
    logger = log.NewWithOptions(os.Stderr, log.Options{ Prefix: "[ad]", Level: log.DebugLevel })
)

type Controller struct { service Service }

func NewController() *Controller { return &Controller{service: NewService()} }

func (c *Controller) RegisterRoutes(router chi.Router) {
    router.Route("/ad", func(r chi.Router) {
        r.Post("/landing", c.TrackLanding)
    })
    logger.Info("╔═════ Ad")
    logger.Info("║   POST /landing  (public)")
    logger.Info("╚═════")
}

type trackLandingReq struct {
    Code string `json:"code"`
}

func (c *Controller) TrackLanding(w http.ResponseWriter, r *http.Request) {
    // Code can come from query (?code=XXX) or JSON body {code}
    code := r.URL.Query().Get("code")
    if code == "" {
        var body trackLandingReq
        _ = json.NewDecoder(r.Body).Decode(&body)
        code = body.Code
    }

    // Extract IP (consider proxies)
    ip := r.Header.Get("X-Forwarded-For")
    if ip == "" {
        ip = r.Header.Get("CF-Connecting-IP")
    }
    if ip == "" {
        ip = r.RemoteAddr
    }
    ua := r.UserAgent()
    referer := r.Referer()

    if err := c.service.TrackLanding(r.Context(), code, ip, ua, referer); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    w.WriteHeader(http.StatusNoContent)
}

