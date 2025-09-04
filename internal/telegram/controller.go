package telegram

import (
    "context"
    "encoding/json"
    "io"
    "net/http"
    "os"
    "time"

    "github.com/charmbracelet/log"
    "github.com/go-chi/chi/v5"

    "github.com/jourloy/nutri-backend/internal/auth"
    "github.com/jourloy/nutri-backend/internal/lib"
)

var (
    logger = log.NewWithOptions(os.Stderr, log.Options{
        Prefix: "[tlgm]",
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
    router.Route("/telegram", func(r chi.Router) {
        // Creates (or returns existing) ticket for current user.
        r.Post("/ticket", c.CreateTicket)
        // Link by token: public endpoint for bot/backend-to-backend.
        r.Post("/link", c.LinkByToken)
        // Returns current user's telegram profile.
        r.Get("/", c.GetMe)
        // Update notify flags for current user's profile.
        r.Patch("/notify", c.UpdateNotify)
        // Returns public telegram info (id, username, avatar) by user id.
        r.Get("/public/{userId}", c.GetPublicByUserId)
        // Proxies telegram avatar file by user id without exposing bot token.
        r.Get("/avatar/{userId}", c.GetAvatarByUserId)
        // Delete current user's telegram profile.
        r.Delete("/", c.DeleteMe)
    })

    logger.Info("╔═════ Telegram")
    logger.Info("║   POST /ticket")
    logger.Info("║   POST /link")
    logger.Info("║    GET /")
    logger.Info("║  PATCH /notify")
    logger.Info("║    GET /public/{userId}")
    logger.Info("║    GET /avatar/{userId}")
    logger.Info("║ DELETE /")
    logger.Info("╚═════")
}

// CreateTicket creates a telegram profile (if not exists) and returns token.
func (c *Controller) CreateTicket(w http.ResponseWriter, r *http.Request) {
    u, ok := auth.UserFromContext(r.Context())
    if !ok {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }

    tp, err := c.service.CreateTicket(context.Background(), u.Id)
    if err != nil {
        logger.Error("create ticket failed", "err", err)
        http.Error(w, "create ticket failed", http.StatusBadRequest)
        return
    }

    w.WriteHeader(http.StatusCreated)
    _ = json.NewEncoder(w).Encode(map[string]any{
        "token": tp.Token,
        "profile": tp,
    })
}

// LinkByToken updates telegram fields by provided token.
func (c *Controller) LinkByToken(w http.ResponseWriter, r *http.Request) {
    var in LinkRequest
    if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    if in.Token == "" {
        http.Error(w, "token is required", http.StatusBadRequest)
        return
    }

    tp, err := c.service.LinkByToken(context.Background(), in)
    if err != nil {
        logger.Error("link by token failed", "err", err)
        http.Error(w, "link failed", http.StatusBadRequest)
        return
    }

    w.WriteHeader(http.StatusOK)
    _ = json.NewEncoder(w).Encode(tp)
}

// GetMe returns current user's telegram profile if exists.
func (c *Controller) GetMe(w http.ResponseWriter, r *http.Request) {
    u, ok := auth.UserFromContext(r.Context())
    if !ok {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }

    tp, err := c.service.GetByUserId(context.Background(), u.Id)
    if err != nil {
        logger.Error("get me failed", "err", err)
        http.Error(w, "not found", http.StatusNotFound)
        return
    }

    w.WriteHeader(http.StatusOK)
    _ = json.NewEncoder(w).Encode(tp)
}

// GetPublicByUserId returns id, username and avatar by user id.
func (c *Controller) GetPublicByUserId(w http.ResponseWriter, r *http.Request) {
    userId := chi.URLParam(r, "userId")
    if userId == "" {
        http.Error(w, "userId is required", http.StatusBadRequest)
        return
    }

    pub, err := c.service.GetPublicByUserId(context.Background(), userId)
    if err != nil {
        logger.Error("get public by user failed", "err", err)
        http.Error(w, "not found", http.StatusNotFound)
        return
    }

    w.WriteHeader(http.StatusOK)
    _ = json.NewEncoder(w).Encode(pub)
}

// GetAvatarByUserId streams telegram avatar image by user id (using stored file_path) without exposing bot token.
func (c *Controller) GetAvatarByUserId(w http.ResponseWriter, r *http.Request) {
    userId := chi.URLParam(r, "userId")
    if userId == "" {
        http.Error(w, "userId is required", http.StatusBadRequest)
        return
    }
    if lib.Config.TelegramToken == "" {
        http.Error(w, "telegram not configured", http.StatusNotImplemented)
        return
    }

    ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
    defer cancel()

    pub, err := c.service.GetPublicByUserId(ctx, userId)
    if err != nil || pub == nil || pub.TelegramAvatar == nil || *pub.TelegramAvatar == "" {
        http.Error(w, "avatar not found", http.StatusNotFound)
        return
    }

    url := "https://api.telegram.org/file/bot" + lib.Config.TelegramToken + "/" + *pub.TelegramAvatar

    req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    if err != nil {
        http.Error(w, "failed to build request", http.StatusInternalServerError)
        return
    }
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        http.Error(w, "failed to fetch file", http.StatusBadGateway)
        return
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        http.Error(w, "failed to fetch file", http.StatusBadGateway)
        return
    }

    // Pass through content type and caching headers conservatively
    if ct := resp.Header.Get("Content-Type"); ct != "" {
        w.Header().Set("Content-Type", ct)
    } else {
        w.Header().Set("Content-Type", "application/octet-stream")
    }
    // Cache for 1 day (avatars change rarely and we refresh daily)
    w.Header().Set("Cache-Control", "public, max-age=86400")

    w.WriteHeader(http.StatusOK)
    _, _ = io.Copy(w, resp.Body)
}

// DeleteMe removes current user's telegram profile.
func (c *Controller) DeleteMe(w http.ResponseWriter, r *http.Request) {
    u, ok := auth.UserFromContext(r.Context())
    if !ok {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }

    if err := c.service.DeleteByUserId(context.Background(), u.Id); err != nil {
        logger.Error("delete failed", "err", err)
        http.Error(w, "delete failed", http.StatusBadRequest)
        return
    }

    w.WriteHeader(http.StatusOK)
}

// UpdateNotify allows partial editing of notify_* flags for current user's telegram profile.
func (c *Controller) UpdateNotify(w http.ResponseWriter, r *http.Request) {
    u, ok := auth.UserFromContext(r.Context())
    if !ok {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }

    var upd NotifyUpdate
    if err := json.NewDecoder(r.Body).Decode(&upd); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    tp, err := c.service.UpdateNotifyByUserId(context.Background(), u.Id, upd)
    if err != nil {
        logger.Error("update notify failed", "err", err)
        http.Error(w, "update failed", http.StatusBadRequest)
        return
    }

    w.WriteHeader(http.StatusOK)
    _ = json.NewEncoder(w).Encode(tp)
}
