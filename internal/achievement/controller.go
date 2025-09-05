package achievement

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
    logger = log.NewWithOptions(os.Stderr, log.Options{ Prefix: "[achv]", Level: log.DebugLevel })
)

type Controller struct { service Service }

func NewController() *Controller { return &Controller{service: NewService()} }

func (c *Controller) RegisterRoutes(router chi.Router) {
    router.Route("/achievement", func(r chi.Router) {
        // Public/user endpoints (auth middleware supplies user context)
        r.Get("/all", c.ListAll)
        r.Get("/my", c.ListMine)
        r.Post("/evaluate", c.Evaluate)

        // Admin endpoints
        r.Post("/", c.Create)
        r.Put("/", c.Update)
        r.Delete("/{id}", c.Delete)

        r.Get("/categories", c.GetCategories)
        r.Post("/category", c.CreateCategory)
        r.Put("/category", c.UpdateCategory)
        r.Delete("/category/{id}", c.DeleteCategory)
    })

    logger.Info("╔═════ Achievement")
    logger.Info("║   GET  /all")
    logger.Info("║   GET  /my")
    logger.Info("║   POST /evaluate")
    logger.Info("║   POST /")
    logger.Info("║   PUT  /")
    logger.Info("║   DELETE /{id}")
    logger.Info("║   GET  /categories")
    logger.Info("║   POST /category")
    logger.Info("║   PUT  /category")
    logger.Info("║   DELETE /category/{id}")
    logger.Info("╚═════")
}

// ===== User endpoints =====
func (c *Controller) ListAll(w http.ResponseWriter, r *http.Request) {
    u, ok := auth.UserFromContext(r.Context())
    if !ok { http.Error(w, "unauthorized", http.StatusUnauthorized); return }
    res, err := c.service.ListForUser(context.Background(), u.Id)
    if err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
    w.WriteHeader(http.StatusOK)
    _ = json.NewEncoder(w).Encode(res)
}

func (c *Controller) ListMine(w http.ResponseWriter, r *http.Request) {
    u, ok := auth.UserFromContext(r.Context())
    if !ok { http.Error(w, "unauthorized", http.StatusUnauthorized); return }
    res, err := c.service.ListUserUnlocked(context.Background(), u.Id)
    if err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
    w.WriteHeader(http.StatusOK)
    _ = json.NewEncoder(w).Encode(res)
}

func (c *Controller) Evaluate(w http.ResponseWriter, r *http.Request) {
    u, ok := auth.UserFromContext(r.Context())
    if !ok { http.Error(w, "unauthorized", http.StatusUnauthorized); return }
    res, err := c.service.EvaluateUser(context.Background(), u.Id)
    if err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
    w.WriteHeader(http.StatusOK)
    _ = json.NewEncoder(w).Encode(res)
}

// ===== Admin CRUD =====
func (c *Controller) Create(w http.ResponseWriter, r *http.Request) {
    u, ok := auth.UserFromContext(r.Context())
    if !ok { http.Error(w, "unauthorized", http.StatusUnauthorized); return }
    if !u.IsAdmin { http.Error(w, "forbidden", http.StatusForbidden); return }
    var a Achievement
    if err := json.NewDecoder(r.Body).Decode(&a); err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
    resp, err := c.service.CreateAchievement(context.Background(), a)
    if err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
    w.WriteHeader(http.StatusCreated)
    _ = json.NewEncoder(w).Encode(resp)
}

func (c *Controller) Update(w http.ResponseWriter, r *http.Request) {
    u, ok := auth.UserFromContext(r.Context())
    if !ok { http.Error(w, "unauthorized", http.StatusUnauthorized); return }
    if !u.IsAdmin { http.Error(w, "forbidden", http.StatusForbidden); return }
    var a Achievement
    if err := json.NewDecoder(r.Body).Decode(&a); err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
    resp, err := c.service.UpdateAchievement(context.Background(), a)
    if err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
    w.WriteHeader(http.StatusOK)
    _ = json.NewEncoder(w).Encode(resp)
}

func (c *Controller) Delete(w http.ResponseWriter, r *http.Request) {
    u, ok := auth.UserFromContext(r.Context())
    if !ok { http.Error(w, "unauthorized", http.StatusUnauthorized); return }
    if !u.IsAdmin { http.Error(w, "forbidden", http.StatusForbidden); return }
    idStr := chi.URLParam(r, "id")
    if idStr == "" { http.Error(w, "missing id", http.StatusBadRequest); return }
    id, err := strconv.ParseInt(idStr, 10, 64)
    if err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
    if err := c.service.DeleteAchievement(context.Background(), id); err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
    w.WriteHeader(http.StatusOK)
}

// Category admin
func (c *Controller) GetCategories(w http.ResponseWriter, r *http.Request) {
    _, ok := auth.UserFromContext(r.Context())
    if !ok { http.Error(w, "unauthorized", http.StatusUnauthorized); return }
    res, err := c.service.GetCategories(context.Background())
    if err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
    w.WriteHeader(http.StatusOK)
    _ = json.NewEncoder(w).Encode(res)
}

func (c *Controller) CreateCategory(w http.ResponseWriter, r *http.Request) {
    u, ok := auth.UserFromContext(r.Context())
    if !ok { http.Error(w, "unauthorized", http.StatusUnauthorized); return }
    if !u.IsAdmin { http.Error(w, "forbidden", http.StatusForbidden); return }
    var cat Category
    if err := json.NewDecoder(r.Body).Decode(&cat); err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
    resp, err := c.service.CreateCategory(context.Background(), cat)
    if err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
    w.WriteHeader(http.StatusCreated)
    _ = json.NewEncoder(w).Encode(resp)
}

func (c *Controller) UpdateCategory(w http.ResponseWriter, r *http.Request) {
    u, ok := auth.UserFromContext(r.Context())
    if !ok { http.Error(w, "unauthorized", http.StatusUnauthorized); return }
    if !u.IsAdmin { http.Error(w, "forbidden", http.StatusForbidden); return }
    var cat Category
    if err := json.NewDecoder(r.Body).Decode(&cat); err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
    resp, err := c.service.UpdateCategory(context.Background(), cat)
    if err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
    w.WriteHeader(http.StatusOK)
    _ = json.NewEncoder(w).Encode(resp)
}

func (c *Controller) DeleteCategory(w http.ResponseWriter, r *http.Request) {
    u, ok := auth.UserFromContext(r.Context())
    if !ok { http.Error(w, "unauthorized", http.StatusUnauthorized); return }
    if !u.IsAdmin { http.Error(w, "forbidden", http.StatusForbidden); return }
    idStr := chi.URLParam(r, "id")
    if idStr == "" { http.Error(w, "missing id", http.StatusBadRequest); return }
    id, err := strconv.ParseInt(idStr, 10, 64)
    if err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
    if err := c.service.DeleteCategory(context.Background(), id); err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
    w.WriteHeader(http.StatusOK)
}

