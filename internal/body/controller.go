package body

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/charmbracelet/log"
	"github.com/go-chi/chi/v5"

	"github.com/jourloy/nutri-backend/internal/auth"
	"github.com/jourloy/nutri-backend/pkg/timeutil"
)

var (
	logger = log.NewWithOptions(os.Stderr, log.Options{Prefix: "[body]", Level: log.DebugLevel})
)

type Controller struct{ service Service }

func NewController() *Controller { return &Controller{service: NewService()} }

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
		// BMI calculation
		r.Get("/bmi", c.GetBMI)
		// workout
		r.Post("/workout", c.CreateWorkout)
		r.Put("/workout", c.UpdateWorkout)
		r.Delete("/workout/{id}", c.DeleteWorkout)
		r.Get("/workout", c.GetWorkouts)
		r.Get("/workout/daily", c.GetWorkoutDailySummary)
		// cycle
		r.Get("/cycle/catalog", c.GetCycleCatalog)
		r.Get("/cycle/summary", c.GetCycleSummary)
		r.Get("/cycle/timeline", c.GetCycleTimeline)
		r.Get("/cycle/day", c.GetCycleDayLogs)
		r.Put("/cycle/day", c.UpsertCycleDay)
		r.Post("/cycle/seed", c.SeedCycle)
		r.Post("/cycle/start", c.StartCycle)
		r.Post("/cycle/stop", c.StopCycle)
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
	logger.Info("║    GET /bmi")
	logger.Info("║   POST /workout")
	logger.Info("║    PUT /workout")
	logger.Info("║ DELETE /workout/{id}")
	logger.Info("║    GET /workout?from=&to=")
	logger.Info("║    GET /workout/daily?date=")
	logger.Info("║    GET /cycle/catalog")
	logger.Info("║    GET /cycle/summary?date=")
	logger.Info("║    GET /cycle/timeline?from=&to=&date=")
	logger.Info("║    GET /cycle/day?from=&to=")
	logger.Info("║    PUT /cycle/day")
	logger.Info("║   POST /cycle/seed")
	logger.Info("║   POST /cycle/start")
	logger.Info("║   POST /cycle/stop")
	logger.Info("╚═════")
}

// ===== Weights =====
func (c *Controller) CreateWeight(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var body struct {
		Value    float64 `json:"value"`
		LoggedAt *string `json:"loggedAt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	when := time.Now()
	if body.LoggedAt != nil && *body.LoggedAt != "" {
		if t, err := time.Parse("2006-01-02", *body.LoggedAt); err == nil {
			when = t
		}
	}
	res, err := c.service.CreateWeight(context.Background(), WeightCreate{UserId: u.Id, Value: body.Value, LoggedAt: when})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(res)
}

func (c *Controller) UpdateWeight(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var body struct {
		Id       int64   `json:"id"`
		Value    float64 `json:"value"`
		LoggedAt *string `json:"loggedAt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	when := time.Now()
	if body.LoggedAt != nil && *body.LoggedAt != "" {
		if t, err := time.Parse("2006-01-02", *body.LoggedAt); err == nil {
			when = t
		}
	}
	res, err := c.service.UpdateWeight(context.Background(), Weight{Id: body.Id, UserId: u.Id, Value: body.Value, LoggedAt: when})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(res)
}

func (c *Controller) DeleteWeight(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := c.service.DeleteWeight(context.Background(), id, u.Id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (c *Controller) GetWeights(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var from, to *time.Time
	if s := r.URL.Query().Get("from"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			from = &t
		}
	}
	if s := r.URL.Query().Get("to"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			to = &t
		}
	}
	res, err := c.service.GetWeights(context.Background(), u.Id, from, to)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(res)
}

func (c *Controller) GetLatestWeight(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	res, err := c.service.GetLatestWeight(context.Background(), u.Id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(res)
}

// ===== Measurements =====
func (c *Controller) CreateMeasurement(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var body struct {
		Chest    *float64 `json:"chest"`
		Waist    *float64 `json:"waist"`
		Hips     *float64 `json:"hips"`
		LoggedAt *string  `json:"loggedAt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if body.Chest == nil && body.Waist == nil && body.Hips == nil {
		http.Error(w, "at least one of chest/waist/hips required", http.StatusBadRequest)
		return
	}
	when := time.Now()
	if body.LoggedAt != nil && *body.LoggedAt != "" {
		if t, err := time.Parse("2006-01-02", *body.LoggedAt); err == nil {
			when = t
		}
	}
	res, err := c.service.CreateMeasurement(context.Background(), MeasurementCreate{UserId: u.Id, Chest: body.Chest, Waist: body.Waist, Hips: body.Hips, LoggedAt: when})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(res)
}

func (c *Controller) UpdateMeasurement(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var body struct {
		Id       int64    `json:"id"`
		Chest    *float64 `json:"chest"`
		Waist    *float64 `json:"waist"`
		Hips     *float64 `json:"hips"`
		LoggedAt *string  `json:"loggedAt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	when := time.Now()
	if body.LoggedAt != nil && *body.LoggedAt != "" {
		if t, err := time.Parse("2006-01-02", *body.LoggedAt); err == nil {
			when = t
		}
	}
	res, err := c.service.UpdateMeasurement(context.Background(), Measurement{Id: body.Id, UserId: u.Id, Chest: body.Chest, Waist: body.Waist, Hips: body.Hips, LoggedAt: when})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(res)
}

func (c *Controller) DeleteMeasurement(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := c.service.DeleteMeasurement(context.Background(), id, u.Id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (c *Controller) GetMeasurements(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var from, to *time.Time
	if s := r.URL.Query().Get("from"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			from = &t
		}
	}
	if s := r.URL.Query().Get("to"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			to = &t
		}
	}
	res, err := c.service.GetMeasurements(context.Background(), u.Id, from, to)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(res)
}

func (c *Controller) GetLatestMeasurement(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	res, err := c.service.GetLatestMeasurement(context.Background(), u.Id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(res)
}

// ===== Plateau =====
func (c *Controller) GetPlateau(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	res, err := c.service.EvaluatePlateau(context.Background(), u.Id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(res)
}

func (c *Controller) EvaluatePlateau(w http.ResponseWriter, r *http.Request) {
	c.GetPlateau(w, r)
}

// ===== Activity CRUD =====
func (c *Controller) CreateActivity(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var body struct {
		Steps    *int    `json:"steps"`
		SleepMin *int    `json:"sleepMin"`
		LoggedAt *string `json:"loggedAt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	when := time.Now()
	if body.LoggedAt != nil && *body.LoggedAt != "" {
		if t, err := time.Parse("2006-01-02", *body.LoggedAt); err == nil {
			when = t
		}
	}
	res, err := c.service.CreateActivity(context.Background(), ActivityCreate{UserId: u.Id, Steps: body.Steps, SleepMin: body.SleepMin, LoggedAt: when})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(res)
}

func (c *Controller) UpdateActivity(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var body struct {
		Id       int64   `json:"id"`
		Steps    *int    `json:"steps"`
		SleepMin *int    `json:"sleepMin"`
		LoggedAt *string `json:"loggedAt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	when := time.Now()
	if body.LoggedAt != nil && *body.LoggedAt != "" {
		if t, err := time.Parse("2006-01-02", *body.LoggedAt); err == nil {
			when = t
		}
	}
	res, err := c.service.UpdateActivity(context.Background(), Activity{Id: body.Id, UserId: u.Id, Steps: body.Steps, SleepMin: body.SleepMin, LoggedAt: when})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(res)
}

func (c *Controller) DeleteActivity(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := c.service.DeleteActivity(context.Background(), id, u.Id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (c *Controller) GetActivity(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var from, to *time.Time
	if s := r.URL.Query().Get("from"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			from = &t
		}
	}
	if s := r.URL.Query().Get("to"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			to = &t
		}
	}
	res, err := c.service.GetActivity(context.Background(), u.Id, from, to)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(res)
}

func (c *Controller) GetPlateauHistory(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var from, to *time.Time
	if s := r.URL.Query().Get("from"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			from = &t
		}
	}
	if s := r.URL.Query().Get("to"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			to = &t
		}
	}
	res, err := c.service.GetPlateauHistory(context.Background(), u.Id, from, to)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(res)
}

// ===== BMI =====
func (c *Controller) GetBMI(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	res, err := c.service.CalculateBMI(context.Background(), u.Id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(res)
}

// ===== Workout =====
func (c *Controller) CreateWorkout(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var body struct {
		LoggedAt       *string `json:"loggedAt"`
		DurationMin    int     `json:"durationMin"`
		WorkoutType    string  `json:"workoutType"`
		Intensity      *string `json:"intensity"`
		CaloriesBurned *int    `json:"caloriesBurned"`
		Note           *string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	today, err := currentDateForUser(u.Timezone)
	if err != nil {
		http.Error(w, "failed to resolve user date", http.StatusBadRequest)
		return
	}
	loggedAt := today
	if body.LoggedAt != nil && *body.LoggedAt != "" {
		parsed, err := parseDate(*body.LoggedAt)
		if err != nil {
			http.Error(w, "invalid loggedAt format, expected YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		loggedAt = parsed
	}
	if loggedAt.After(today) {
		http.Error(w, ErrWorkoutFutureDate.Error(), http.StatusBadRequest)
		return
	}

	res, err := c.service.CreateWorkout(context.Background(), WorkoutCreate{
		UserId:         u.Id,
		LoggedAt:       loggedAt,
		DurationMin:    body.DurationMin,
		WorkoutType:    body.WorkoutType,
		Intensity:      body.Intensity,
		CaloriesBurned: body.CaloriesBurned,
		Note:           body.Note,
	}, today)
	if err != nil {
		c.writeWorkoutError(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(res)
}

func (c *Controller) UpdateWorkout(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var body struct {
		Id             int64   `json:"id"`
		LoggedAt       *string `json:"loggedAt"`
		DurationMin    int     `json:"durationMin"`
		WorkoutType    string  `json:"workoutType"`
		Intensity      *string `json:"intensity"`
		CaloriesBurned *int    `json:"caloriesBurned"`
		Note           *string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	today, err := currentDateForUser(u.Timezone)
	if err != nil {
		http.Error(w, "failed to resolve user date", http.StatusBadRequest)
		return
	}
	loggedAt := today
	if body.LoggedAt != nil && *body.LoggedAt != "" {
		parsed, err := parseDate(*body.LoggedAt)
		if err != nil {
			http.Error(w, "invalid loggedAt format, expected YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		loggedAt = parsed
	}
	if loggedAt.After(today) {
		http.Error(w, ErrWorkoutFutureDate.Error(), http.StatusBadRequest)
		return
	}

	res, err := c.service.UpdateWorkout(context.Background(), body.Id, WorkoutCreate{
		UserId:         u.Id,
		LoggedAt:       loggedAt,
		DurationMin:    body.DurationMin,
		WorkoutType:    body.WorkoutType,
		Intensity:      body.Intensity,
		CaloriesBurned: body.CaloriesBurned,
		Note:           body.Note,
	}, today)
	if err != nil {
		c.writeWorkoutError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(res)
}

func (c *Controller) DeleteWorkout(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := c.service.DeleteWorkout(context.Background(), id, u.Id); err != nil {
		c.writeWorkoutError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (c *Controller) GetWorkouts(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var from *time.Time
	if fromRaw := r.URL.Query().Get("from"); fromRaw != "" {
		parsed, err := parseDate(fromRaw)
		if err != nil {
			http.Error(w, "invalid from format, expected YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		from = &parsed
	}
	var to *time.Time
	if toRaw := r.URL.Query().Get("to"); toRaw != "" {
		parsed, err := parseDate(toRaw)
		if err != nil {
			http.Error(w, "invalid to format, expected YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		to = &parsed
	}

	res, err := c.service.GetWorkouts(context.Background(), u.Id, from, to)
	if err != nil {
		c.writeWorkoutError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(res)
}

func (c *Controller) GetWorkoutDailySummary(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	today, err := currentDateForUser(u.Timezone)
	if err != nil {
		http.Error(w, "failed to resolve user date", http.StatusBadRequest)
		return
	}

	date := today
	if dateRaw := r.URL.Query().Get("date"); dateRaw != "" {
		parsed, err := parseDate(dateRaw)
		if err != nil {
			http.Error(w, "invalid date format, expected YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		date = parsed
	}

	res, err := c.service.GetWorkoutDailySummary(context.Background(), u.Id, date, today)
	if err != nil {
		c.writeWorkoutError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(res)
}

// ===== Cycle =====
func (c *Controller) GetCycleCatalog(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	res, err := c.service.GetCycleCatalog(context.Background(), u.Id)
	if err != nil {
		c.writeCycleError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(res)
}

func (c *Controller) GetCycleSummary(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var date *time.Time
	if dateRaw := r.URL.Query().Get("date"); dateRaw != "" {
		parsed, err := parseDate(dateRaw)
		if err != nil {
			http.Error(w, "invalid date format, expected YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		date = &parsed
	}

	res, err := c.service.GetCycleSummary(context.Background(), u.Id, date)
	if err != nil {
		c.writeCycleError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(res)
}

func (c *Controller) GetCycleTimeline(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var from *time.Time
	if fromRaw := r.URL.Query().Get("from"); fromRaw != "" {
		parsed, err := parseDate(fromRaw)
		if err != nil {
			http.Error(w, "invalid from format, expected YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		from = &parsed
	}

	var to *time.Time
	if toRaw := r.URL.Query().Get("to"); toRaw != "" {
		parsed, err := parseDate(toRaw)
		if err != nil {
			http.Error(w, "invalid to format, expected YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		to = &parsed
	}

	var date *time.Time
	if dateRaw := r.URL.Query().Get("date"); dateRaw != "" {
		parsed, err := parseDate(dateRaw)
		if err != nil {
			http.Error(w, "invalid date format, expected YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		date = &parsed
	}

	res, err := c.service.GetCycleTimeline(context.Background(), u.Id, from, to, date)
	if err != nil {
		c.writeCycleError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(res)
}

func (c *Controller) GetCycleDayLogs(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var from *time.Time
	if fromRaw := r.URL.Query().Get("from"); fromRaw != "" {
		parsed, err := parseDate(fromRaw)
		if err != nil {
			http.Error(w, "invalid from format, expected YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		from = &parsed
	}

	var to *time.Time
	if toRaw := r.URL.Query().Get("to"); toRaw != "" {
		parsed, err := parseDate(toRaw)
		if err != nil {
			http.Error(w, "invalid to format, expected YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		to = &parsed
	}

	if from != nil && to != nil && from.After(*to) {
		http.Error(w, "from must be <= to", http.StatusBadRequest)
		return
	}

	res, err := c.service.GetCycleDayLogs(context.Background(), u.Id, from, to)
	if err != nil {
		c.writeCycleError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(res)
}

func (c *Controller) UpsertCycleDay(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var body struct {
		LoggedAt      *string `json:"loggedAt"`
		FlowIntensity *string `json:"flowIntensity"`
		Note          *string `json:"note"`
		Events        []struct {
			EventCategory string  `json:"eventCategory"`
			EventCode     string  `json:"eventCode"`
			Quantity      int     `json:"quantity"`
			Intensity     *string `json:"intensity"`
		} `json:"events"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	loggedAt := time.Now()
	if body.LoggedAt != nil && *body.LoggedAt != "" {
		parsed, err := parseDate(*body.LoggedAt)
		if err != nil {
			http.Error(w, "invalid loggedAt format, expected YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		loggedAt = parsed
	}

	events := make([]CycleDayEventInput, 0, len(body.Events))
	for _, event := range body.Events {
		events = append(events, CycleDayEventInput{
			EventCategory: event.EventCategory,
			EventCode:     event.EventCode,
			Quantity:      event.Quantity,
			Intensity:     event.Intensity,
		})
	}

	res, err := c.service.UpsertCycleDay(context.Background(), u.Id, CycleDayUpsertInput{
		LoggedAt:      loggedAt,
		FlowIntensity: body.FlowIntensity,
		Note:          body.Note,
		Events:        events,
	})
	if err != nil {
		c.writeCycleError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(res)
}

func (c *Controller) StartCycle(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var body struct {
		LoggedAt *string `json:"loggedAt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	loggedAt := time.Now()
	if body.LoggedAt != nil && *body.LoggedAt != "" {
		parsed, err := parseDate(*body.LoggedAt)
		if err != nil {
			http.Error(w, "invalid loggedAt format, expected YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		loggedAt = parsed
	}

	res, err := c.service.StartCycle(context.Background(), u.Id, loggedAt)
	if err != nil {
		c.writeCycleError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(res)
}

func (c *Controller) SeedCycle(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var body struct {
		Mode     string  `json:"mode"`
		LoggedAt *string `json:"loggedAt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if body.LoggedAt == nil || *body.LoggedAt == "" {
		http.Error(w, "invalid loggedAt format, expected YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	loggedAt, err := parseDate(*body.LoggedAt)
	if err != nil {
		http.Error(w, "invalid loggedAt format, expected YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	res, err := c.service.SeedCycle(context.Background(), u.Id, CycleSeedInput{
		Mode:     body.Mode,
		LoggedAt: loggedAt,
	})
	if err != nil {
		c.writeCycleError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(res)
}

func (c *Controller) StopCycle(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var body struct {
		LoggedAt *string `json:"loggedAt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	loggedAt := time.Now()
	if body.LoggedAt != nil && *body.LoggedAt != "" {
		parsed, err := parseDate(*body.LoggedAt)
		if err != nil {
			http.Error(w, "invalid loggedAt format, expected YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		loggedAt = parsed
	}

	res, err := c.service.StopCycle(context.Background(), u.Id, loggedAt)
	if err != nil {
		c.writeCycleError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(res)
}

func parseDate(value string) (time.Time, error) {
	return time.Parse("2006-01-02", value)
}

func currentDateForUser(timezone *string) (time.Time, error) {
	today := timeutil.CurrentDateForTimezone(timeutil.GetTimezoneOrDefault(timezone))
	return parseDate(today)
}

func (c *Controller) writeWorkoutError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrWorkoutInputInvalid), errors.Is(err, ErrWorkoutFutureDate):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}

func (c *Controller) writeCycleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrCycleForbidden):
		http.Error(w, err.Error(), http.StatusForbidden)
	case errors.Is(err, ErrCycleNoActive), errors.Is(err, ErrCycleDateInvalid), errors.Is(err, ErrCycleInputInvalid):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}
