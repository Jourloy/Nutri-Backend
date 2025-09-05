package body

import (
    "context"
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
    logger = log.NewWithOptions(os.Stderr, log.Options{ Prefix: "[body]", Level: log.DebugLevel })
)

type Controller struct { service Service }

func NewController() *Controller { return &Controller{ service: NewService() } }

func (c *Controller) RegisterRoutes(router chi.Router) {
    router.Route("/body", func(r chi.Router) {
        // weights
        r.Post("/weight", c.CreateWeight)
        r.Put("/weight", c.UpdateWeight)
        r.Delete("/weight/{id}", c.DeleteWeight)
        r.Get("/weights", c.GetWeights)
        r.Get("/weight/latest", c.GetLatestWeight)
        // measurements
        r.Post("/measure", c.CreateMeasurement)
        r.Put("/measure", c.UpdateMeasurement)
        r.Delete("/measure/{id}", c.DeleteMeasurement)
        r.Get("/measures", c.GetMeasurements)
        r.Get("/measure/latest", c.GetLatestMeasurement)
        // plateau
        r.Get("/plateau", c.GetPlateau)
        r.Post("/plateau/evaluate", c.EvaluatePlateau)

        // activity (steps/sleep)
        r.Post("/activity", c.CreateActivity)
        r.Put("/activity", c.UpdateActivity)
        r.Delete("/activity/{id}", c.DeleteActivity)
        r.Get("/activity", c.GetActivity)
        // plateau history
        r.Get("/plateau/history", c.GetPlateauHistory)
    })

    logger.Info("╔═════ BodyTracking")
    logger.Info("║   POST /weight")
    logger.Info("║    PUT /weight")
    logger.Info("║ DELETE /weight/{id}")
    logger.Info("║    GET /weights?from=&to=")
    logger.Info("║    GET /weight/latest")
    logger.Info("║   POST /measure")
    logger.Info("║    PUT /measure")
    logger.Info("║ DELETE /measure/{id}")
    logger.Info("║    GET /measures?from=&to=")
    logger.Info("║    GET /measure/latest")
    logger.Info("║    GET /plateau")
    logger.Info("║   POST /plateau/evaluate")
    logger.Info("║   POST /activity")
    logger.Info("║    PUT /activity")
    logger.Info("║ DELETE /activity/{id}")
    logger.Info("║    GET /activity?from=&to=")
    logger.Info("║    GET /plateau/history?from=&to=")
    logger.Info("╚═════")
}

// ===== Weights =====
func (c *Controller) CreateWeight(w http.ResponseWriter, r *http.Request) {
    u, ok := auth.UserFromContext(r.Context()); if !ok { http.Error(w, "unauthorized", http.StatusUnauthorized); return }
    var body struct { Value float64 `json:"value"`; LoggedAt *string `json:"loggedAt"` }
    if err := json.NewDecoder(r.Body).Decode(&body); err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
    when := time.Now()
    if body.LoggedAt != nil && *body.LoggedAt != "" { if t, err := time.Parse("2006-01-02", *body.LoggedAt); err == nil { when = t } }
    res, err := c.service.CreateWeight(context.Background(), WeightCreate{UserId: u.Id, Value: body.Value, LoggedAt: when})
    if err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
    w.WriteHeader(http.StatusCreated); _ = json.NewEncoder(w).Encode(res)
}

func (c *Controller) UpdateWeight(w http.ResponseWriter, r *http.Request) {
    u, ok := auth.UserFromContext(r.Context()); if !ok { http.Error(w, "unauthorized", http.StatusUnauthorized); return }
    var body struct { Id int64 `json:"id"`; Value float64 `json:"value"`; LoggedAt *string `json:"loggedAt"` }
    if err := json.NewDecoder(r.Body).Decode(&body); err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
    when := time.Now()
    if body.LoggedAt != nil && *body.LoggedAt != "" { if t, err := time.Parse("2006-01-02", *body.LoggedAt); err == nil { when = t } }
    res, err := c.service.UpdateWeight(context.Background(), Weight{Id: body.Id, UserId: u.Id, Value: body.Value, LoggedAt: when})
    if err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
    w.WriteHeader(http.StatusOK); _ = json.NewEncoder(w).Encode(res)
}

func (c *Controller) DeleteWeight(w http.ResponseWriter, r *http.Request) {
    u, ok := auth.UserFromContext(r.Context()); if !ok { http.Error(w, "unauthorized", http.StatusUnauthorized); return }
    idStr := chi.URLParam(r, "id"); if idStr == "" { http.Error(w, "missing id", http.StatusBadRequest); return }
    id, err := strconv.ParseInt(idStr, 10, 64); if err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
    if err := c.service.DeleteWeight(context.Background(), id, u.Id); err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
    w.WriteHeader(http.StatusOK)
}

func (c *Controller) GetWeights(w http.ResponseWriter, r *http.Request) {
    u, ok := auth.UserFromContext(r.Context()); if !ok { http.Error(w, "unauthorized", http.StatusUnauthorized); return }
    var from, to *time.Time
    if s := r.URL.Query().Get("from"); s != "" { if t, err := time.Parse("2006-01-02", s); err == nil { from = &t } }
    if s := r.URL.Query().Get("to"); s != "" { if t, err := time.Parse("2006-01-02", s); err == nil { to = &t } }
    res, err := c.service.GetWeights(context.Background(), u.Id, from, to)
    if err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
    w.WriteHeader(http.StatusOK); _ = json.NewEncoder(w).Encode(res)
}

func (c *Controller) GetLatestWeight(w http.ResponseWriter, r *http.Request) {
    u, ok := auth.UserFromContext(r.Context()); if !ok { http.Error(w, "unauthorized", http.StatusUnauthorized); return }
    res, err := c.service.GetLatestWeight(context.Background(), u.Id)
    if err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
    w.WriteHeader(http.StatusOK); _ = json.NewEncoder(w).Encode(res)
}

// ===== Measurements =====
func (c *Controller) CreateMeasurement(w http.ResponseWriter, r *http.Request) {
    u, ok := auth.UserFromContext(r.Context()); if !ok { http.Error(w, "unauthorized", http.StatusUnauthorized); return }
    var body struct { Chest *float64 `json:"chest"`; Waist *float64 `json:"waist"`; Hips *float64 `json:"hips"`; LoggedAt *string `json:"loggedAt"` }
    if err := json.NewDecoder(r.Body).Decode(&body); err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
    if body.Chest == nil && body.Waist == nil && body.Hips == nil { http.Error(w, "at least one of chest/waist/hips required", http.StatusBadRequest); return }
    when := time.Now()
    if body.LoggedAt != nil && *body.LoggedAt != "" { if t, err := time.Parse("2006-01-02", *body.LoggedAt); err == nil { when = t } }
    res, err := c.service.CreateMeasurement(context.Background(), MeasurementCreate{UserId: u.Id, Chest: body.Chest, Waist: body.Waist, Hips: body.Hips, LoggedAt: when})
    if err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
    w.WriteHeader(http.StatusCreated); _ = json.NewEncoder(w).Encode(res)
}

func (c *Controller) UpdateMeasurement(w http.ResponseWriter, r *http.Request) {
    u, ok := auth.UserFromContext(r.Context()); if !ok { http.Error(w, "unauthorized", http.StatusUnauthorized); return }
    var body struct { Id int64 `json:"id"`; Chest *float64 `json:"chest"`; Waist *float64 `json:"waist"`; Hips *float64 `json:"hips"`; LoggedAt *string `json:"loggedAt"` }
    if err := json.NewDecoder(r.Body).Decode(&body); err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
    when := time.Now()
    if body.LoggedAt != nil && *body.LoggedAt != "" { if t, err := time.Parse("2006-01-02", *body.LoggedAt); err == nil { when = t } }
    res, err := c.service.UpdateMeasurement(context.Background(), Measurement{Id: body.Id, UserId: u.Id, Chest: body.Chest, Waist: body.Waist, Hips: body.Hips, LoggedAt: when})
    if err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
    w.WriteHeader(http.StatusOK); _ = json.NewEncoder(w).Encode(res)
}

func (c *Controller) DeleteMeasurement(w http.ResponseWriter, r *http.Request) {
    u, ok := auth.UserFromContext(r.Context()); if !ok { http.Error(w, "unauthorized", http.StatusUnauthorized); return }
    idStr := chi.URLParam(r, "id"); if idStr == "" { http.Error(w, "missing id", http.StatusBadRequest); return }
    id, err := strconv.ParseInt(idStr, 10, 64); if err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
    if err := c.service.DeleteMeasurement(context.Background(), id, u.Id); err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
    w.WriteHeader(http.StatusOK)
}

func (c *Controller) GetMeasurements(w http.ResponseWriter, r *http.Request) {
    u, ok := auth.UserFromContext(r.Context()); if !ok { http.Error(w, "unauthorized", http.StatusUnauthorized); return }
    var from, to *time.Time
    if s := r.URL.Query().Get("from"); s != "" { if t, err := time.Parse("2006-01-02", s); err == nil { from = &t } }
    if s := r.URL.Query().Get("to"); s != "" { if t, err := time.Parse("2006-01-02", s); err == nil { to = &t } }
    res, err := c.service.GetMeasurements(context.Background(), u.Id, from, to)
    if err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
    w.WriteHeader(http.StatusOK); _ = json.NewEncoder(w).Encode(res)
}

func (c *Controller) GetLatestMeasurement(w http.ResponseWriter, r *http.Request) {
    u, ok := auth.UserFromContext(r.Context()); if !ok { http.Error(w, "unauthorized", http.StatusUnauthorized); return }
    res, err := c.service.GetLatestMeasurement(context.Background(), u.Id)
    if err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
    w.WriteHeader(http.StatusOK); _ = json.NewEncoder(w).Encode(res)
}

// ===== Plateau =====
func (c *Controller) GetPlateau(w http.ResponseWriter, r *http.Request) {
    u, ok := auth.UserFromContext(r.Context()); if !ok { http.Error(w, "unauthorized", http.StatusUnauthorized); return }
    res, err := c.service.EvaluatePlateau(context.Background(), u.Id)
    if err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
    w.WriteHeader(http.StatusOK); _ = json.NewEncoder(w).Encode(res)
}

func (c *Controller) EvaluatePlateau(w http.ResponseWriter, r *http.Request) {
    c.GetPlateau(w, r)
}

// ===== Activity CRUD =====
func (c *Controller) CreateActivity(w http.ResponseWriter, r *http.Request) {
    u, ok := auth.UserFromContext(r.Context()); if !ok { http.Error(w, "unauthorized", http.StatusUnauthorized); return }
    var body struct{ Steps *int `json:"steps"`; SleepMin *int `json:"sleepMin"`; LoggedAt *string `json:"loggedAt"` }
    if err := json.NewDecoder(r.Body).Decode(&body); err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
    when := time.Now(); if body.LoggedAt != nil && *body.LoggedAt != "" { if t, err := time.Parse("2006-01-02", *body.LoggedAt); err == nil { when = t } }
    res, err := c.service.CreateActivity(context.Background(), ActivityCreate{UserId: u.Id, Steps: body.Steps, SleepMin: body.SleepMin, LoggedAt: when})
    if err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
    w.WriteHeader(http.StatusCreated); _ = json.NewEncoder(w).Encode(res)
}

func (c *Controller) UpdateActivity(w http.ResponseWriter, r *http.Request) {
    u, ok := auth.UserFromContext(r.Context()); if !ok { http.Error(w, "unauthorized", http.StatusUnauthorized); return }
    var body struct{ Id int64 `json:"id"`; Steps *int `json:"steps"`; SleepMin *int `json:"sleepMin"`; LoggedAt *string `json:"loggedAt"` }
    if err := json.NewDecoder(r.Body).Decode(&body); err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
    when := time.Now(); if body.LoggedAt != nil && *body.LoggedAt != "" { if t, err := time.Parse("2006-01-02", *body.LoggedAt); err == nil { when = t } }
    res, err := c.service.UpdateActivity(context.Background(), Activity{Id: body.Id, UserId: u.Id, Steps: body.Steps, SleepMin: body.SleepMin, LoggedAt: when})
    if err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
    w.WriteHeader(http.StatusOK); _ = json.NewEncoder(w).Encode(res)
}

func (c *Controller) DeleteActivity(w http.ResponseWriter, r *http.Request) {
    u, ok := auth.UserFromContext(r.Context()); if !ok { http.Error(w, "unauthorized", http.StatusUnauthorized); return }
    idStr := chi.URLParam(r, "id"); if idStr == "" { http.Error(w, "missing id", http.StatusBadRequest); return }
    id, err := strconv.ParseInt(idStr, 10, 64); if err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
    if err := c.service.DeleteActivity(context.Background(), id, u.Id); err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
    w.WriteHeader(http.StatusOK)
}

func (c *Controller) GetActivity(w http.ResponseWriter, r *http.Request) {
    u, ok := auth.UserFromContext(r.Context()); if !ok { http.Error(w, "unauthorized", http.StatusUnauthorized); return }
    var from, to *time.Time
    if s := r.URL.Query().Get("from"); s != "" { if t, err := time.Parse("2006-01-02", s); err == nil { from = &t } }
    if s := r.URL.Query().Get("to"); s != "" { if t, err := time.Parse("2006-01-02", s); err == nil { to = &t } }
    res, err := c.service.GetActivity(context.Background(), u.Id, from, to)
    if err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
    w.WriteHeader(http.StatusOK); _ = json.NewEncoder(w).Encode(res)
}

func (c *Controller) GetPlateauHistory(w http.ResponseWriter, r *http.Request) {
    u, ok := auth.UserFromContext(r.Context()); if !ok { http.Error(w, "unauthorized", http.StatusUnauthorized); return }
    var from, to *time.Time
    if s := r.URL.Query().Get("from"); s != "" { if t, err := time.Parse("2006-01-02", s); err == nil { from = &t } }
    if s := r.URL.Query().Get("to"); s != "" { if t, err := time.Parse("2006-01-02", s); err == nil { to = &t } }
    res, err := c.service.GetPlateauHistory(context.Background(), u.Id, from, to)
    if err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
    w.WriteHeader(http.StatusOK); _ = json.NewEncoder(w).Encode(res)
}
