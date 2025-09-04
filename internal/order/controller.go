package order

import (
    "context"
    "encoding/json"
    "io"
    "net/http"
    "os"
    "strconv"
    "strings"

    "github.com/charmbracelet/log"
    "github.com/go-chi/chi/v5"

    "github.com/jourloy/nutri-backend/internal/auth"
)

var (
    logger = log.NewWithOptions(os.Stderr, log.Options{Prefix: "[orde]", Level: log.DebugLevel})
)

type Controller struct{ service Service }

func NewController() *Controller { return &Controller{service: NewService()} }

func (c *Controller) RegisterRoutes(r chi.Router) {
    r.Route("/order", func(r chi.Router) {
        r.Post("/init", c.Init)
        r.Post("/webhook/tbank", c.WebhookTBank)
        r.Get("/all", c.GetAll)
        r.Delete("/{id}", c.Delete)
        r.Post("/ensure-start", c.EnsureStart)
    })
    logger.Info("╔═════ Order")
    logger.Info("║   GET /all")
    logger.Info("║  POST /init")
    logger.Info("║  POST /webhook/tbank")
    logger.Info("║  POST /ensure-start")
    logger.Info("║ DELETE /{id}")
    logger.Info("╚═════")
}

func (c *Controller) Init(w http.ResponseWriter, r *http.Request) {
    u, ok := auth.UserFromContext(r.Context())
    if !ok {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }

    var p InitPayload
    if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    if p.PlanId <= 0 {
        http.Error(w, "invalid planId", http.StatusBadRequest)
        return
    }

    res, err := c.service.Init(context.Background(), u.Id, p.PlanId, p.ReturnURL)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    w.WriteHeader(http.StatusOK)
    _ = json.NewEncoder(w).Encode(res)
}

func (c *Controller) WebhookTBank(w http.ResponseWriter, r *http.Request) {
    // Read raw body for signature verification
    raw, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    // Parse into generic map to compute Token
    var mm map[string]any
    if err := json.Unmarshal(raw, &mm); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    recvToken, _ := mm["Token"].(string)
    if recvToken == "" {
        http.Error(w, "no token", http.StatusForbidden)
        return
    }
    // Compute expected token using terminal password
    secret := os.Getenv("TBANK_TERMINAL_PASSWORD")
    exp := signToken(secret, mm)
    if strings.ToLower(recvToken) != strings.ToLower(exp) {
        http.Error(w, "invalid token", http.StatusForbidden)
        return
    }
    // Decode to typed payload
    var p TBankWebhook
    if err := json.Unmarshal(raw, &p); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    if err := c.service.HandleTBankWebhook(context.Background(), p); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    w.WriteHeader(http.StatusOK)
}

func (c *Controller) GetAll(w http.ResponseWriter, r *http.Request) {
    u, ok := auth.UserFromContext(r.Context())
    if !ok {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }
    userID := u.Id
    if u.IsAdmin {
        if q := r.URL.Query().Get("userId"); q != "" {
            userID = q
        } else {
            userID = ""
        }
    }
    res, err := c.service.List(context.Background(), userID, u.IsAdmin)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    w.WriteHeader(http.StatusOK)
    _ = json.NewEncoder(w).Encode(res)
}

func (c *Controller) Delete(w http.ResponseWriter, r *http.Request) {
    u, ok := auth.UserFromContext(r.Context())
    if !ok {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }
    idStr := chi.URLParam(r, "id")
    id, err := strconv.ParseInt(idStr, 10, 64)
    if err != nil {
        http.Error(w, "invalid id", http.StatusBadRequest)
        return
    }
    userID := u.Id
    if u.IsAdmin {
        if q := r.URL.Query().Get("userId"); q != "" {
            userID = q
        }
    }
    if err := c.service.Delete(context.Background(), id, userID, u.IsAdmin); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    w.WriteHeader(http.StatusOK)
}

func (c *Controller) EnsureStart(w http.ResponseWriter, r *http.Request) {
    u, ok := auth.UserFromContext(r.Context())
    if !ok {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }

    _, created, err := c.service.EnsureStart(context.Background(), u.Id)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    w.WriteHeader(http.StatusOK)
    _ = json.NewEncoder(w).Encode(map[string]any{"created": created})
}
