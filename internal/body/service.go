package body

import (
	"context"
	"errors"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/jourloy/nutri02/internal/database"
)

const (
	defaultCycleLengthDays   = 28
	defaultPeriodLengthDays  = 5
	periodCaloriesMultiplier = 1.08
	defaultOvulationOffset   = 14
	fertileWindowLeadDays    = 5

	cyclePhaseMenstrual  = "menstrual"
	cyclePhaseFollicular = "follicular"
	cyclePhaseOvulation  = "ovulation"
	cyclePhaseLuteal     = "luteal"

	cycleSourceRecorded  = "recorded"
	cycleSourcePredicted = "predicted"

	cyclePeriodNone      = "none"
	cyclePeriodRecorded  = "recorded"
	cyclePeriodPredicted = "predicted"

	cycleSeedModeStarted = "started"
	cycleSeedModeEnded   = "ended"

	dropletStateFilling  = "filling"
	dropletStateEmptying = "emptying"
	dropletStateSteady   = "steady"

	workoutWaterBonusPerMinute = 10
	workoutWaterLimitMaxMl     = 4500
	defaultWaterLimitMl        = 3000
	maxWorkoutDurationMin      = 600
	maxWorkoutNoteLength       = 500
)

var (
	ErrCycleForbidden    = errors.New("cycle access forbidden")
	ErrCycleNoActive     = errors.New("no active cycle")
	ErrCycleDateInvalid  = errors.New("invalid cycle date")
	ErrCycleInputInvalid = errors.New("invalid cycle input")

	ErrWorkoutInputInvalid = errors.New("invalid workout input")
	ErrWorkoutFutureDate   = errors.New("future workout date is not allowed")

	allowedFlowIntensity = map[string]struct{}{
		"spotting": {},
		"light":    {},
		"medium":   {},
		"heavy":    {},
	}

	allowedEventIntensity = map[string]struct{}{
		"low":    {},
		"medium": {},
		"high":   {},
	}

	allowedWorkoutTypes = map[string]struct{}{
		"strength": {},
		"cardio":   {},
		"hiit":     {},
		"running":  {},
		"walking":  {},
		"yoga":     {},
		"pilates":  {},
		"swimming": {},
		"other":    {},
	}

	cycleCatalog = []CycleCatalogCategory{
		{
			Category: "sex",
			Events: []CycleCatalogEvent{
				{Code: "sex_protected"},
				{Code: "sex_unprotected"},
				{Code: "sex_oral"},
				{Code: "sex_anal"},
				{Code: "masturbation"},
				{Code: "orgasm"},
				{Code: "libido_high"},
				{Code: "libido_low"},
			},
		},
		{
			Category: "activity",
			Events: []CycleCatalogEvent{
				{Code: "workout_strength"},
				{Code: "workout_cardio"},
				{Code: "workout_hiit"},
				{Code: "workout_yoga"},
				{Code: "workout_pilates"},
				{Code: "workout_running"},
				{Code: "workout_walking"},
				{Code: "workout_swimming"},
			},
		},
		{
			Category: "symptoms",
			Events: []CycleCatalogEvent{
				{Code: "cramps"},
				{Code: "headache"},
				{Code: "back_pain"},
				{Code: "bloating"},
				{Code: "fatigue"},
				{Code: "acne"},
				{Code: "breast_tenderness"},
				{Code: "nausea"},
			},
		},
		{
			Category: "mood",
			Events: []CycleCatalogEvent{
				{Code: "mood_calm"},
				{Code: "mood_happy"},
				{Code: "mood_irritable"},
				{Code: "mood_anxious"},
				{Code: "mood_sad"},
			},
		},
		{
			Category: "sleep",
			Events: []CycleCatalogEvent{
				{Code: "sleep_good"},
				{Code: "sleep_poor"},
				{Code: "insomnia"},
				{Code: "daytime_sleepiness"},
			},
		},
		{
			Category: "discharge",
			Events: []CycleCatalogEvent{
				{Code: "discharge_none"},
				{Code: "discharge_sticky"},
				{Code: "discharge_creamy"},
				{Code: "discharge_watery"},
				{Code: "discharge_eggwhite"},
				{Code: "spotting"},
			},
		},
		{
			Category: "lifestyle",
			Events: []CycleCatalogEvent{
				{Code: "stress_high"},
				{Code: "alcohol"},
				{Code: "caffeine_high"},
				{Code: "cravings_sweet"},
				{Code: "cravings_salty"},
			},
		},
		{
			Category: "health",
			Events: []CycleCatalogEvent{
				{Code: "painkiller_taken"},
				{Code: "hormonal_medication"},
				{Code: "pregnancy_test"},
			},
		},
	}

	cycleCatalogMap = buildCycleCatalogMap(cycleCatalog)
)

type Service interface {
	// weights
	CreateWeight(ctx context.Context, w WeightCreate) (*Weight, error)
	UpdateWeight(ctx context.Context, w Weight) (*Weight, error)
	DeleteWeight(ctx context.Context, id int64, userId string) error
	GetWeights(ctx context.Context, userId string, from, to *time.Time) ([]Weight, error)
	GetLatestWeight(ctx context.Context, userId string) (*Weight, error)
	// measurements
	CreateMeasurement(ctx context.Context, m MeasurementCreate) (*Measurement, error)
	UpdateMeasurement(ctx context.Context, m Measurement) (*Measurement, error)
	DeleteMeasurement(ctx context.Context, id int64, userId string) error
	GetMeasurements(ctx context.Context, userId string, from, to *time.Time) ([]Measurement, error)
	GetLatestMeasurement(ctx context.Context, userId string) (*Measurement, error)
	// analytics
	EvaluatePlateau(ctx context.Context, userId string) (*PlateauResult, error)
	// activity
	CreateActivity(ctx context.Context, a ActivityCreate) (*Activity, error)
	UpdateActivity(ctx context.Context, a Activity) (*Activity, error)
	DeleteActivity(ctx context.Context, id int64, userId string) error
	GetActivity(ctx context.Context, userId string, from, to *time.Time) ([]Activity, error)
	// workouts
	CreateWorkout(ctx context.Context, w WorkoutCreate, today time.Time) (*Workout, error)
	UpdateWorkout(ctx context.Context, id int64, w WorkoutCreate, today time.Time) (*Workout, error)
	DeleteWorkout(ctx context.Context, id int64, userId string) error
	GetWorkouts(ctx context.Context, userId string, from, to *time.Time) ([]Workout, error)
	GetWorkoutDailySummary(ctx context.Context, userId string, date, today time.Time) (*WorkoutDailySummary, error)
	// plateau history
	GetPlateauHistory(ctx context.Context, userId string, from, to *time.Time) ([]PlateauEvent, error)
	// BMI calculation
	CalculateBMI(ctx context.Context, userId string) (*BMIResult, error)
	// cycle tracking
	GetCycleCatalog(ctx context.Context, userId string) ([]CycleCatalogCategory, error)
	GetCycleSummary(ctx context.Context, userId string, at *time.Time) (*CycleSummary, error)
	GetCycleTimeline(ctx context.Context, userId string, from, to, at *time.Time) (*CycleTimeline, error)
	GetCycleDayLogs(ctx context.Context, userId string, from, to *time.Time) ([]CycleDayLog, error)
	UpsertCycleDay(ctx context.Context, userId string, input CycleDayUpsertInput) (*CycleDayLog, error)
	SeedCycle(ctx context.Context, userId string, input CycleSeedInput) (*CycleSummary, error)
	StartCycle(ctx context.Context, userId string, at time.Time) (*CycleSummary, error)
	StopCycle(ctx context.Context, userId string, at time.Time) (*CycleSummary, error)
}

type service struct {
	repo Repository
	db   *sqlx.DB
}

type cycleWindow struct {
	Start       time.Time
	NextStart   time.Time
	StartSource string
	ActualCycle *Cycle
}

type cycleDayState struct {
	Date                time.Time
	Phase               string
	PhaseSource         string
	PeriodStatus        string
	IsFertileWindow     bool
	IsOvulationDay      bool
	DropletFillRatio    float64
	DropletState        string
	CycleDayNumber      int
	DaysUntilNextPeriod int
	DaysUntilPhaseShift int
	CurrentCycleStart   time.Time
	CurrentCycleEnd     *time.Time
	PredictedPeriodEnd  time.Time
	PredictedNextStart  time.Time
	PredictedOvulation  time.Time
	FertileWindowStart  time.Time
	FertileWindowEnd    time.Time
	IsRecordedPeriod    bool
}

func NewService() Service { return &service{repo: NewRepository(), db: database.Database} }

// passthrough
func (s *service) CreateWeight(ctx context.Context, w WeightCreate) (*Weight, error) {
	return s.repo.CreateWeight(ctx, w)
}
func (s *service) UpdateWeight(ctx context.Context, w Weight) (*Weight, error) {
	return s.repo.UpdateWeight(ctx, w)
}
func (s *service) DeleteWeight(ctx context.Context, id int64, userId string) error {
	return s.repo.DeleteWeight(ctx, id, userId)
}
func (s *service) GetWeights(ctx context.Context, userId string, from, to *time.Time) ([]Weight, error) {
	return s.repo.GetWeights(ctx, userId, from, to)
}
func (s *service) GetLatestWeight(ctx context.Context, userId string) (*Weight, error) {
	return s.repo.GetLatestWeight(ctx, userId)
}
func (s *service) CreateMeasurement(ctx context.Context, m MeasurementCreate) (*Measurement, error) {
	return s.repo.CreateMeasurement(ctx, m)
}
func (s *service) UpdateMeasurement(ctx context.Context, m Measurement) (*Measurement, error) {
	return s.repo.UpdateMeasurement(ctx, m)
}
func (s *service) DeleteMeasurement(ctx context.Context, id int64, userId string) error {
	return s.repo.DeleteMeasurement(ctx, id, userId)
}
func (s *service) GetMeasurements(ctx context.Context, userId string, from, to *time.Time) ([]Measurement, error) {
	return s.repo.GetMeasurements(ctx, userId, from, to)
}
func (s *service) GetLatestMeasurement(ctx context.Context, userId string) (*Measurement, error) {
	return s.repo.GetLatestMeasurement(ctx, userId)
}

// activity passthrough
func (s *service) CreateActivity(ctx context.Context, a ActivityCreate) (*Activity, error) {
	return s.repo.CreateActivity(ctx, a)
}
func (s *service) UpdateActivity(ctx context.Context, a Activity) (*Activity, error) {
	return s.repo.UpdateActivity(ctx, a)
}
func (s *service) DeleteActivity(ctx context.Context, id int64, userId string) error {
	return s.repo.DeleteActivity(ctx, id, userId)
}
func (s *service) GetActivity(ctx context.Context, userId string, from, to *time.Time) ([]Activity, error) {
	return s.repo.GetActivity(ctx, userId, from, to)
}

func (s *service) CreateWorkout(ctx context.Context, w WorkoutCreate, today time.Time) (*Workout, error) {
	normalized, err := normalizeWorkoutInput(w)
	if err != nil {
		return nil, err
	}
	if err := validateWorkoutDateNotFuture(normalized.LoggedAt, today); err != nil {
		return nil, err
	}
	return s.repo.CreateWorkout(ctx, normalized)
}

func (s *service) UpdateWorkout(ctx context.Context, id int64, w WorkoutCreate, today time.Time) (*Workout, error) {
	if id <= 0 {
		return nil, ErrWorkoutInputInvalid
	}
	normalized, err := normalizeWorkoutInput(w)
	if err != nil {
		return nil, err
	}
	if err := validateWorkoutDateNotFuture(normalized.LoggedAt, today); err != nil {
		return nil, err
	}
	return s.repo.UpdateWorkout(ctx, normalized, id)
}

func (s *service) DeleteWorkout(ctx context.Context, id int64, userId string) error {
	if id <= 0 {
		return ErrWorkoutInputInvalid
	}
	return s.repo.DeleteWorkout(ctx, id, userId)
}

func (s *service) GetWorkouts(ctx context.Context, userId string, from, to *time.Time) ([]Workout, error) {
	var fromDate *time.Time
	var toDate *time.Time
	if from != nil {
		v := dateOnly(*from)
		fromDate = &v
	}
	if to != nil {
		v := dateOnly(*to)
		toDate = &v
	}
	if fromDate != nil && toDate != nil && fromDate.After(*toDate) {
		return nil, ErrWorkoutInputInvalid
	}
	return s.repo.GetWorkouts(ctx, userId, fromDate, toDate)
}

func (s *service) GetWorkoutDailySummary(ctx context.Context, userId string, date, today time.Time) (*WorkoutDailySummary, error) {
	date = dateOnly(date)
	today = dateOnly(today)

	baseLimit, err := s.repo.GetLatestWaterLimit(ctx, userId)
	if err != nil {
		return nil, err
	}
	if baseLimit <= 0 {
		baseLimit = defaultWaterLimitMl
	}

	totalDuration, err := s.repo.GetWorkoutTotalDurationByDate(ctx, userId, date)
	if err != nil {
		return nil, err
	}

	applied := date.Equal(today)
	waterBonus := 0
	if applied {
		waterBonus = totalDuration * workoutWaterBonusPerMinute
	}

	effectiveLimit := baseLimit + waterBonus
	if effectiveLimit > workoutWaterLimitMaxMl {
		effectiveLimit = workoutWaterLimitMaxMl
	}

	return &WorkoutDailySummary{
		Date:                  date.Format("2006-01-02"),
		TotalDurationMin:      totalDuration,
		BaseWaterLimitMl:      baseLimit,
		WaterBonusMl:          waterBonus,
		EffectiveWaterLimitMl: effectiveLimit,
		MaxWaterLimitMl:       workoutWaterLimitMaxMl,
		Applied:               applied,
	}, nil
}

func (s *service) GetPlateauHistory(ctx context.Context, userId string, from, to *time.Time) ([]PlateauEvent, error) {
	return s.repo.GetPlateauHistory(ctx, userId, from, to)
}

// ===== Plateau evaluation =====
func (s *service) EvaluatePlateau(ctx context.Context, userId string) (*PlateauResult, error) {
	// window
	windowDays := 21
	end := time.Now().Truncate(24 * time.Hour)
	start := end.AddDate(0, 0, -windowDays+1)

	// fetch weights (avg per day)
	// body_weights logged_at is a date already
	ws, err := s.repo.GetWeights(ctx, userId, &start, &end)
	if err != nil {
		return nil, err
	}
	// build series by day ascending
	type dp struct {
		d time.Time
		v float64
	}
	m := map[string]float64{}
	for _, w := range ws {
		m[w.LoggedAt.Format("2006-01-02")] = w.Value
	}
	series := make([]dp, 0, windowDays)
	for d := 0; d < windowDays; d++ {
		day := start.AddDate(0, 0, d)
		if v, ok := m[day.Format("2006-01-02")]; ok {
			series = append(series, dp{d: day, v: v})
		}
	}
	if len(series) < 7 {
		// not enough data
		res := &PlateauResult{IsPlateau: false, Goal: "unknown", WindowStart: start.Format("2006-01-02"), WindowEnd: end.Format("2006-01-02"), WindowDays: windowDays, DaysWithWeight: len(series), Reason: "bodyTracking.plateau.reason.insufficientData"}
		return res, nil
	}

	// EWMA smoothing (lambda=0.2)
	lambda := 0.2
	sm := make([]float64, len(series))
	sm[0] = series[0].v
	for i := 1; i < len(series); i++ {
		sm[i] = lambda*series[i].v + (1-lambda)*sm[i-1]
	}

	// OLS slope over indexes 0..n-1
	n := float64(len(sm))
	var sumX, sumY, sumXY, sumXX float64
	for i, y := range sm {
		x := float64(i)
		sumX += x
		sumY += y
		sumXY += x * y
		sumXX += x * x
	}
	denom := n*sumXX - sumX*sumX
	if denom == 0 {
		denom = 1
	}
	slopePerDay := (n*sumXY - sumX*sumY) / denom // kg/day
	meanW := sumY / n
	slopeWeeklyPct := (slopePerDay / meanW) * 7 * 100.0
	deltaKg := sm[len(sm)-1] - sm[0]

	// Fetch fit profile for goal and targets
	var goal string
	var targetCalories, profileWeight float64
	_ = s.db.GetContext(ctx, &goal, `SELECT goal FROM fit_profiles WHERE user_id=$1 ORDER BY created_at DESC LIMIT 1`, userId)
	_ = s.db.GetContext(ctx, &targetCalories, `SELECT calories::float FROM fit_profiles WHERE user_id=$1 ORDER BY created_at DESC LIMIT 1`, userId)
	_ = s.db.GetContext(ctx, &profileWeight, `SELECT weight::float FROM fit_profiles WHERE user_id=$1 ORDER BY created_at DESC LIMIT 1`, userId)
	if goal == "" {
		goal = "unknown"
	}

	// Compliance: calories +/- 10%, protein >= 1.6g/kg
	dailyCals, _ := s.repo.GetDailyCalories(ctx, userId, start, end)
	dailyProt, _ := s.repo.GetDailyProtein(ctx, userId, start, end)
	dailySteps, _ := s.repo.GetDailySteps(ctx, userId, start, end)
	dailySleep, _ := s.repo.GetDailySleepMin(ctx, userId, start, end)
	periodDays, err := s.repo.GetPeriodDays(ctx, userId, start, end)
	if err != nil {
		return nil, err
	}
	calsGood := 0
	protGood := 0
	proteinTarget := 1.6 * profileWeight
	// iterate days in window
	for d := 0; d < windowDays; d++ {
		day := start.AddDate(0, 0, d).Format("2006-01-02")
		targetForDay := targetCalories
		if periodDays[day] {
			targetForDay = effectiveCaloriesTarget(targetCalories)
		}
		lower := 0.9 * targetForDay
		upper := 1.1 * targetForDay
		if v, ok := dailyCals[day]; ok && targetForDay > 0 && v >= lower && v <= upper {
			calsGood++
		}
		if v, ok := dailyProt[day]; ok && proteinTarget > 0 && v >= proteinTarget {
			protGood++
		}
	}
	// steps & sleep averages across the window
	var stepsSum int
	var sleepMinSum int
	for d := 0; d < windowDays; d++ {
		day := start.AddDate(0, 0, d).Format("2006-01-02")
		if v, ok := dailySteps[day]; ok {
			stepsSum += v
		}
		if v, ok := dailySleep[day]; ok {
			sleepMinSum += v
		}
	}
	stepsAvg := float64(stepsSum) / float64(windowDays)
	sleepAvgHours := float64(sleepMinSum) / float64(windowDays*60)
	// steps target from fit profile
	var stepsTarget int
	_ = s.db.GetContext(ctx, &stepsTarget, `SELECT steps_target FROM fit_profiles WHERE user_id=$1 ORDER BY created_at DESC LIMIT 1`, userId)
	if stepsTarget <= 0 {
		stepsTarget = 8000
	}

	sufficient := (calsGood >= 14) && (protGood >= 14) && (stepsAvg >= 0.8*float64(stepsTarget)) && (sleepAvgHours >= 6.0)

	// Plateau rules
	isPlateau := false
	reason := ""
	// base thresholds
	thrPct := 0.05    // per week
	smallDelta := 0.3 // kg

	switch goal {
	case "lose", "fat_loss", "weight_loss":
		if (slopeWeeklyPct > -thrPct) || (math.Abs(deltaKg) < smallDelta) {
			isPlateau = true
			reason = "bodyTracking.plateau.reason.weakWeightLoss"
		}
	case "gain", "muscle_gain", "bulk":
		if (slopeWeeklyPct < thrPct) || (math.Abs(deltaKg) < smallDelta) {
			isPlateau = true
			reason = "bodyTracking.plateau.reason.weakWeightGain"
		}
	default:
		// if unknown goal — decide only by flat trend
		if math.Abs(slopeWeeklyPct) < thrPct && math.Abs(deltaKg) < smallDelta {
			isPlateau = true
			reason = "bodyTracking.plateau.reason.flatTrend"
		}
	}

	if isPlateau && !sufficient {
		// compliance not sufficient -> do not flag plateau strictly
		isPlateau = false
		if reason == "" {
			reason = "bodyTracking.plateau.reason.poorAdherence"
		} else {
			reason += "; bodyTracking.plateau.reason.poorAdherence"
		}
	}

	// persist event
	_, _ = s.db.ExecContext(ctx, `
        INSERT INTO body_plateau_events (user_id, window_start, window_end, goal, slope_weekly_pct, delta_kg, days_with_weight, calories_good_days, protein_good_days, window_days, is_plateau, reason)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		userId, start, end, goal, slopeWeeklyPct, deltaKg, len(series), calsGood, protGood, windowDays, isPlateau, reason,
	)

	res := &PlateauResult{
		IsPlateau:        isPlateau,
		Goal:             goal,
		WindowStart:      start.Format("2006-01-02"),
		WindowEnd:        end.Format("2006-01-02"),
		WindowDays:       windowDays,
		DaysWithWeight:   len(series),
		SlopeWeeklyPct:   slopeWeeklyPct,
		DeltaKg:          deltaKg,
		CaloriesGoodDays: calsGood,
		ProteinGoodDays:  protGood,
		CaloriesTarget:   targetCalories,
		ProteinPerKg:     1.6,
		StepsAvg:         stepsAvg,
		StepsTarget:      stepsTarget,
		SleepAvgHours:    sleepAvgHours,
		Reason:           reason,
	}
	return res, nil
}

// ===== BMI calculation =====
func (s *service) CalculateBMI(ctx context.Context, userId string) (*BMIResult, error) {
	// Get latest weight
	weight, err := s.repo.GetLatestWeight(ctx, userId)
	if err != nil || weight == nil {
		return nil, err
	}

	// Get height from fit profile (in cm)
	var heightCm float64
	err = s.db.GetContext(ctx, &heightCm, `SELECT height::float FROM fit_profiles WHERE user_id=$1 ORDER BY created_at DESC LIMIT 1`, userId)
	if err != nil {
		return nil, err
	}

	if heightCm <= 0 {
		return nil, err
	}

	// Calculate BMI
	heightM := heightCm / 100.0
	bmi := weight.Value / (heightM * heightM)

	// Determine BMI category
	category := ""
	isHealthy := false
	switch {
	case bmi < 18.5:
		category = "bodyTracking.bmi.category.underweight"
	case bmi >= 18.5 && bmi < 25:
		category = "bodyTracking.bmi.category.normal"
		isHealthy = true
	case bmi >= 25 && bmi < 30:
		category = "bodyTracking.bmi.category.overweight"
	case bmi >= 30:
		category = "bodyTracking.bmi.category.obese"
	}

	// Calculate recommended weight range (BMI 18.5-24.9)
	recommendedMinKg := 18.5 * heightM * heightM
	recommendedMaxKg := 24.9 * heightM * heightM

	// Calculate difference from recommended range
	diffFromMin := weight.Value - recommendedMinKg
	diffFromMax := weight.Value - recommendedMaxKg

	return &BMIResult{
		CurrentWeight:    weight.Value,
		Height:           heightCm,
		BMI:              bmi,
		BMICategory:      category,
		RecommendedMinKg: recommendedMinKg,
		RecommendedMaxKg: recommendedMaxKg,
		IsHealthy:        isHealthy,
		DiffFromMin:      diffFromMin,
		DiffFromMax:      diffFromMax,
	}, nil
}

// ===== Cycle tracking =====
func (s *service) GetCycleCatalog(ctx context.Context, userId string) ([]CycleCatalogCategory, error) {
	if err := s.ensureFemaleCycleAccess(ctx, userId); err != nil {
		return nil, err
	}
	return cloneCycleCatalog(cycleCatalog), nil
}

func (s *service) GetCycleSummary(ctx context.Context, userId string, at *time.Time) (*CycleSummary, error) {
	if err := s.ensureFemaleCycleAccess(ctx, userId); err != nil {
		return nil, err
	}

	day := time.Now()
	if at != nil {
		day = *at
	}
	day = dateOnly(day)
	dayKey := day.Format("2006-01-02")

	periodDays, err := s.repo.GetPeriodDays(ctx, userId, day, day)
	if err != nil {
		return nil, err
	}
	isPeriodDay := periodDays[dayKey]

	goals, err := s.repo.GetLatestMacroGoals(ctx, userId)
	if err != nil {
		return nil, err
	}

	cycles, err := s.repo.GetCycles(ctx, userId, nil, nil, 64)
	if err != nil {
		return nil, err
	}

	cycleLen, periodLen, confidence := deriveCycleForecast(cycles)
	currentCycle := findCycleForDay(cycles, day)
	latestStart := latestCycleStart(cycles)

	summary := &CycleSummary{
		Date:             dayKey,
		IsPeriodDay:      isPeriodDay,
		CycleLengthDays:  cycleLen,
		PeriodLengthDays: periodLen,
		Confidence:       confidence,
		BaseGoals:        goals,
		EffectiveGoals:   applyPeriodGoals(goals, isPeriodDay),
	}

	if currentCycle != nil {
		start := currentCycle.StartDate.Format("2006-01-02")
		summary.CurrentCycleStart = &start
		if currentCycle.EndDate != nil {
			end := currentCycle.EndDate.Format("2006-01-02")
			summary.CurrentCycleEnd = &end
		}
		if currentCycle.EndDate == nil {
			predictedPeriodEnd := currentCycle.StartDate.AddDate(0, 0, periodLen-1).Format("2006-01-02")
			summary.PredictedPeriodEnd = &predictedPeriodEnd
		}
	}

	if latestStart != nil {
		nextStart := latestStart.AddDate(0, 0, cycleLen).Format("2006-01-02")
		summary.PredictedNextStart = &nextStart
	}

	return summary, nil
}

func (s *service) GetCycleTimeline(
	ctx context.Context,
	userId string,
	from, to, at *time.Time,
) (*CycleTimeline, error) {
	if err := s.ensureFemaleCycleAccess(ctx, userId); err != nil {
		return nil, err
	}

	selectedDay := time.Now()
	if at != nil {
		selectedDay = *at
	}
	selectedDay = dateOnly(selectedDay)

	rangeStart := selectedDay
	if from != nil {
		rangeStart = dateOnly(*from)
	}
	rangeEnd := selectedDay
	if to != nil {
		rangeEnd = dateOnly(*to)
	}
	if from == nil && to != nil {
		rangeStart = rangeEnd
	}
	if from != nil && to == nil {
		rangeEnd = rangeStart
	}
	if rangeStart.After(rangeEnd) {
		return nil, ErrCycleInputInvalid
	}

	goals, err := s.repo.GetLatestMacroGoals(ctx, userId)
	if err != nil {
		return nil, err
	}

	cycles, err := s.repo.GetCycles(ctx, userId, nil, nil, 128)
	if err != nil {
		return nil, err
	}
	openCycle, err := s.repo.GetOpenCycle(ctx, userId)
	if err != nil {
		return nil, err
	}
	logs, err := s.repo.GetCycleDayLogs(ctx, userId, &rangeStart, &rangeEnd)
	if err != nil {
		return nil, err
	}

	cycleLen, periodLen, confidence := deriveCycleForecast(cycles)
	today := dateOnly(time.Now())
	logMap := make(map[string]CycleDayLog, len(logs))
	for _, logItem := range logs {
		logMap[logItem.LoggedAt] = logItem
	}

	days := make([]CycleTimelineDay, 0, dayDiff(rangeStart, rangeEnd)+1)
	for day := rangeStart; !day.After(rangeEnd); day = day.AddDate(0, 0, 1) {
		state := buildCycleDayState(day, selectedDay, today, cycles, cycleLen, periodLen)
		logItem, hasLog := logMap[day.Format("2006-01-02")]
		days = append(days, CycleTimelineDay{
			Date:            day.Format("2006-01-02"),
			Phase:           state.Phase,
			PhaseSource:     state.PhaseSource,
			PeriodStatus:    state.PeriodStatus,
			IsFertileWindow: state.IsFertileWindow,
			IsOvulationDay:  state.IsOvulationDay,
			HasLog:          hasLog,
			FlowIntensity:   logItem.FlowIntensity,
		})
	}

	selectedState := buildCycleDayState(selectedDay, selectedDay, today, cycles, cycleLen, periodLen)
	var currentCycleStart *string
	if !selectedState.CurrentCycleStart.IsZero() {
		value := selectedState.CurrentCycleStart.Format("2006-01-02")
		currentCycleStart = &value
	}

	var activeCycleStart *string
	if openCycle != nil {
		value := dateOnly(openCycle.StartDate).Format("2006-01-02")
		activeCycleStart = &value
	}

	var currentCycleEnd *string
	if selectedState.CurrentCycleEnd != nil {
		value := selectedState.CurrentCycleEnd.Format("2006-01-02")
		currentCycleEnd = &value
	}

	predictedPeriodEnd := selectedState.PredictedPeriodEnd.Format("2006-01-02")
	predictedNextStart := selectedState.PredictedNextStart.Format("2006-01-02")
	predictedOvulation := selectedState.PredictedOvulation.Format("2006-01-02")
	fertileStart := selectedState.FertileWindowStart.Format("2006-01-02")
	fertileEnd := selectedState.FertileWindowEnd.Format("2006-01-02")

	summary := CycleTimelineSummary{
		CycleSummary: CycleSummary{
			Date:               selectedDay.Format("2006-01-02"),
			IsPeriodDay:        selectedState.IsRecordedPeriod,
			CurrentCycleStart:  currentCycleStart,
			CurrentCycleEnd:    currentCycleEnd,
			CycleLengthDays:    cycleLen,
			PeriodLengthDays:   periodLen,
			PredictedPeriodEnd: &predictedPeriodEnd,
			PredictedNextStart: &predictedNextStart,
			Confidence:         confidence,
			BaseGoals:          goals,
			EffectiveGoals:     applyPeriodGoals(goals, selectedState.IsRecordedPeriod),
		},
		Phase:                  selectedState.Phase,
		PhaseSource:            selectedState.PhaseSource,
		PeriodStatus:           selectedState.PeriodStatus,
		HasCycleSeed:           len(cycles) > 0,
		HasActiveCycle:         openCycle != nil,
		ActiveCycleStart:       activeCycleStart,
		CycleDayNumber:         selectedState.CycleDayNumber,
		PredictedOvulationDate: &predictedOvulation,
		FertileWindowStart:     &fertileStart,
		FertileWindowEnd:       &fertileEnd,
		DropletFillRatio:       selectedState.DropletFillRatio,
		DropletState:           selectedState.DropletState,
		DaysUntilNextPeriod:    selectedState.DaysUntilNextPeriod,
		DaysUntilPhaseChange:   selectedState.DaysUntilPhaseShift,
	}

	return &CycleTimeline{
		Summary: summary,
		Days:    days,
	}, nil
}

func (s *service) GetCycleDayLogs(ctx context.Context, userId string, from, to *time.Time) ([]CycleDayLog, error) {
	if err := s.ensureFemaleCycleAccess(ctx, userId); err != nil {
		return nil, err
	}

	var fromDate *time.Time
	var toDate *time.Time
	if from != nil {
		v := dateOnly(*from)
		fromDate = &v
	}
	if to != nil {
		v := dateOnly(*to)
		toDate = &v
	}
	if fromDate != nil && toDate != nil && fromDate.After(*toDate) {
		return nil, ErrCycleInputInvalid
	}

	return s.repo.GetCycleDayLogs(ctx, userId, fromDate, toDate)
}

func (s *service) UpsertCycleDay(ctx context.Context, userId string, input CycleDayUpsertInput) (*CycleDayLog, error) {
	if err := s.ensureFemaleCycleAccess(ctx, userId); err != nil {
		return nil, err
	}

	normalized, err := normalizeCycleDayInput(input)
	if err != nil {
		return nil, err
	}

	dayLogID, err := s.repo.UpsertCycleDayLog(
		ctx,
		userId,
		normalized.LoggedAt,
		normalized.FlowIntensity,
		normalized.Note,
	)
	if err != nil {
		return nil, err
	}
	if err := s.repo.ReplaceCycleDayEvents(ctx, dayLogID, normalized.Events); err != nil {
		return nil, err
	}

	from := normalized.LoggedAt
	to := normalized.LoggedAt
	logs, err := s.repo.GetCycleDayLogs(ctx, userId, &from, &to)
	if err != nil {
		return nil, err
	}
	if len(logs) == 0 {
		return &CycleDayLog{
			Id:       dayLogID,
			LoggedAt: normalized.LoggedAt.Format("2006-01-02"),
			Events:   []CycleDailyEvent{},
		}, nil
	}
	return &logs[0], nil
}

func (s *service) StartCycle(ctx context.Context, userId string, at time.Time) (*CycleSummary, error) {
	if err := s.ensureFemaleCycleAccess(ctx, userId); err != nil {
		return nil, err
	}

	when := at
	if when.IsZero() {
		when = time.Now()
	}
	when = dateOnly(when)

	if err := s.repo.StartCycle(ctx, userId, when); err != nil {
		return nil, err
	}
	return s.GetCycleSummary(ctx, userId, &when)
}

func (s *service) SeedCycle(ctx context.Context, userId string, input CycleSeedInput) (*CycleSummary, error) {
	if err := s.ensureFemaleCycleAccess(ctx, userId); err != nil {
		return nil, err
	}

	mode, err := normalizeCycleSeedMode(input.Mode)
	if err != nil {
		return nil, err
	}

	when := dateOnly(input.LoggedAt)
	if when.IsZero() {
		when = dateOnly(time.Now())
	}
	if when.After(dateOnly(time.Now())) {
		return nil, ErrCycleDateInvalid
	}

	if mode == cycleSeedModeStarted {
		return s.StartCycle(ctx, userId, when)
	}

	openCycle, err := s.repo.GetOpenCycle(ctx, userId)
	if err != nil {
		return nil, err
	}
	if openCycle != nil {
		return nil, ErrCycleInputInvalid
	}

	cycles, err := s.repo.GetCycles(ctx, userId, nil, nil, 128)
	if err != nil {
		return nil, err
	}
	_, periodLen, _ := deriveCycleForecast(cycles)
	endDate := when.AddDate(0, 0, -1)
	startDate := endDate.AddDate(0, 0, -(maxInt(periodLen, 1) - 1))
	if _, err := s.repo.CreateHistoricalCycle(ctx, userId, startDate, endDate); err != nil {
		return nil, err
	}
	return s.GetCycleSummary(ctx, userId, &when)
}

func (s *service) StopCycle(ctx context.Context, userId string, at time.Time) (*CycleSummary, error) {
	if err := s.ensureFemaleCycleAccess(ctx, userId); err != nil {
		return nil, err
	}

	when := at
	if when.IsZero() {
		when = time.Now()
	}
	when = dateOnly(when)

	openCycle, err := s.repo.GetOpenCycle(ctx, userId)
	if err != nil {
		return nil, err
	}
	if openCycle == nil {
		return nil, ErrCycleNoActive
	}
	startDate := dateOnly(openCycle.StartDate)
	if !when.After(startDate) {
		return nil, ErrCycleDateInvalid
	}

	if _, err := s.repo.StopCycle(ctx, userId, when.AddDate(0, 0, -1)); err != nil {
		return nil, err
	}
	return s.GetCycleSummary(ctx, userId, &when)
}

func (s *service) ensureFemaleCycleAccess(ctx context.Context, userId string) error {
	gender, err := s.repo.GetLatestGender(ctx, userId)
	if err != nil {
		return err
	}
	if strings.ToLower(strings.TrimSpace(gender)) != "female" {
		return ErrCycleForbidden
	}
	return nil
}

func normalizeWorkoutInput(input WorkoutCreate) (WorkoutCreate, error) {
	out := input
	out.LoggedAt = dateOnly(out.LoggedAt)

	if out.DurationMin <= 0 || out.DurationMin > maxWorkoutDurationMin {
		return WorkoutCreate{}, ErrWorkoutInputInvalid
	}

	workoutType := strings.ToLower(strings.TrimSpace(out.WorkoutType))
	if _, ok := allowedWorkoutTypes[workoutType]; !ok {
		return WorkoutCreate{}, ErrWorkoutInputInvalid
	}
	out.WorkoutType = workoutType

	if out.Intensity != nil {
		intensity := strings.ToLower(strings.TrimSpace(*out.Intensity))
		if intensity == "" {
			out.Intensity = nil
		} else {
			if _, ok := allowedEventIntensity[intensity]; !ok {
				return WorkoutCreate{}, ErrWorkoutInputInvalid
			}
			out.Intensity = &intensity
		}
	}

	if out.CaloriesBurned != nil && *out.CaloriesBurned < 0 {
		return WorkoutCreate{}, ErrWorkoutInputInvalid
	}

	if out.Note != nil {
		note := strings.TrimSpace(*out.Note)
		if note == "" {
			out.Note = nil
		} else {
			if len([]rune(note)) > maxWorkoutNoteLength {
				return WorkoutCreate{}, ErrWorkoutInputInvalid
			}
			out.Note = &note
		}
	}

	return out, nil
}

func validateWorkoutDateNotFuture(loggedAt, today time.Time) error {
	if dateOnly(loggedAt).After(dateOnly(today)) {
		return ErrWorkoutFutureDate
	}
	return nil
}

func normalizeCycleDayInput(input CycleDayUpsertInput) (CycleDayUpsertInput, error) {
	loggedAt := input.LoggedAt
	if loggedAt.IsZero() {
		loggedAt = time.Now()
	}
	out := CycleDayUpsertInput{
		LoggedAt: dateOnly(loggedAt),
		Events:   make([]CycleDayEventInput, 0, len(input.Events)),
	}

	if input.FlowIntensity != nil {
		flow := strings.ToLower(strings.TrimSpace(*input.FlowIntensity))
		if flow != "" {
			if _, ok := allowedFlowIntensity[flow]; !ok {
				return CycleDayUpsertInput{}, ErrCycleInputInvalid
			}
			out.FlowIntensity = &flow
		}
	}

	if input.Note != nil {
		note := strings.TrimSpace(*input.Note)
		if note != "" {
			out.Note = &note
		}
	}

	for _, event := range input.Events {
		category := strings.ToLower(strings.TrimSpace(event.EventCategory))
		code := strings.ToLower(strings.TrimSpace(event.EventCode))
		if category == "" || code == "" {
			return CycleDayUpsertInput{}, ErrCycleInputInvalid
		}
		validCodes, ok := cycleCatalogMap[category]
		if !ok {
			return CycleDayUpsertInput{}, ErrCycleInputInvalid
		}
		if _, ok := validCodes[code]; !ok {
			return CycleDayUpsertInput{}, ErrCycleInputInvalid
		}

		quantity := event.Quantity
		if quantity <= 0 {
			quantity = 1
		}

		var intensity *string
		if event.Intensity != nil {
			v := strings.ToLower(strings.TrimSpace(*event.Intensity))
			if v != "" {
				if _, ok := allowedEventIntensity[v]; !ok {
					return CycleDayUpsertInput{}, ErrCycleInputInvalid
				}
				intensity = &v
			}
		}

		out.Events = append(out.Events, CycleDayEventInput{
			EventCategory: category,
			EventCode:     code,
			Quantity:      quantity,
			Intensity:     intensity,
		})
	}

	return out, nil
}

func buildCycleCatalogMap(catalog []CycleCatalogCategory) map[string]map[string]struct{} {
	result := make(map[string]map[string]struct{}, len(catalog))
	for _, category := range catalog {
		events := make(map[string]struct{}, len(category.Events))
		for _, event := range category.Events {
			events[event.Code] = struct{}{}
		}
		result[category.Category] = events
	}
	return result
}

func cloneCycleCatalog(src []CycleCatalogCategory) []CycleCatalogCategory {
	out := make([]CycleCatalogCategory, 0, len(src))
	for _, category := range src {
		copiedEvents := make([]CycleCatalogEvent, len(category.Events))
		copy(copiedEvents, category.Events)
		out = append(out, CycleCatalogCategory{
			Category: category.Category,
			Events:   copiedEvents,
		})
	}
	return out
}

func deriveCycleForecast(cycles []Cycle) (int, int, string) {
	cycleLen := defaultCycleLengthDays
	periodLen := defaultPeriodLengthDays

	if len(cycles) > 0 {
		starts := make([]time.Time, 0, len(cycles))
		for _, cycle := range cycles {
			starts = append(starts, dateOnly(cycle.StartDate))
		}
		sort.Slice(starts, func(i, j int) bool {
			return starts[i].Before(starts[j])
		})

		intervals := make([]int, 0, len(starts)-1)
		for i := 1; i < len(starts); i++ {
			days := int(starts[i].Sub(starts[i-1]).Hours() / 24)
			if days > 0 {
				intervals = append(intervals, days)
			}
		}
		if len(intervals) > 6 {
			intervals = intervals[len(intervals)-6:]
		}
		if len(intervals) > 0 {
			cycleLen = clampInt(medianInt(intervals), 21, 45)
		}

		type duration struct {
			start time.Time
			days  int
		}
		durations := make([]duration, 0, len(cycles))
		for _, cycle := range cycles {
			if cycle.EndDate == nil {
				continue
			}
			endDate := dateOnly(*cycle.EndDate)
			startDate := dateOnly(cycle.StartDate)
			if endDate.Before(startDate) {
				continue
			}
			durations = append(durations, duration{
				start: startDate,
				days:  int(endDate.Sub(startDate).Hours()/24) + 1,
			})
		}
		sort.Slice(durations, func(i, j int) bool {
			return durations[i].start.After(durations[j].start)
		})
		if len(durations) > 6 {
			durations = durations[:6]
		}
		if len(durations) > 0 {
			vals := make([]int, 0, len(durations))
			for _, d := range durations {
				vals = append(vals, d.days)
			}
			periodLen = clampInt(medianInt(vals), 2, 10)
		}
	}

	return cycleLen, periodLen, cycleConfidence(len(cycles))
}

func findCycleForDay(cycles []Cycle, day time.Time) *Cycle {
	for _, cycle := range cycles {
		startDate := dateOnly(cycle.StartDate)
		if cycle.EndDate == nil {
			if !day.Before(startDate) {
				copyCycle := cycle
				return &copyCycle
			}
			continue
		}
		endDate := dateOnly(*cycle.EndDate)
		if !day.Before(startDate) && !day.After(endDate) {
			copyCycle := cycle
			return &copyCycle
		}
	}
	return nil
}

func latestCycleStart(cycles []Cycle) *time.Time {
	if len(cycles) == 0 {
		return nil
	}
	latest := dateOnly(cycles[0].StartDate)
	for i := 1; i < len(cycles); i++ {
		current := dateOnly(cycles[i].StartDate)
		if current.After(latest) {
			latest = current
		}
	}
	return &latest
}

func applyPeriodGoals(base CycleGoals, isPeriodDay bool) CycleGoals {
	if !isPeriodDay {
		return base
	}
	effectiveCalories := effectiveCaloriesTarget(base.Calories)
	if base.Calories <= 0 {
		return CycleGoals{
			Calories: effectiveCalories,
			Protein:  base.Protein,
			Fat:      base.Fat,
			Carbs:    base.Carbs,
		}
	}
	multiplier := effectiveCalories / base.Calories
	return CycleGoals{
		Calories: effectiveCalories,
		Protein:  math.Round(base.Protein * multiplier),
		Fat:      math.Round(base.Fat * multiplier),
		Carbs:    math.Round(base.Carbs * multiplier),
	}
}

func buildCycleDayState(
	day, selectedDay, today time.Time,
	cycles []Cycle,
	cycleLen, periodLen int,
) cycleDayState {
	window := resolveCycleWindow(day, selectedDay, cycles, cycleLen, periodLen)
	cycleSpanDays := maxInt(1, dayDiff(window.Start, window.NextStart))
	effectivePeriodLen := minInt(maxInt(periodLen, 1), cycleSpanDays)
	forecastPeriodEnd := window.Start.AddDate(0, 0, effectivePeriodLen-1)

	actualRecordedEnd := recordedCycleEnd(window.ActualCycle, today)
	var currentCycleEnd *time.Time
	if window.ActualCycle != nil && window.ActualCycle.EndDate != nil {
		end := dateOnly(*window.ActualCycle.EndDate)
		currentCycleEnd = &end
	}
	phasePeriodEnd := forecastPeriodEnd
	if actualRecordedEnd != nil {
		phasePeriodEnd = *actualRecordedEnd
		if phasePeriodEnd.Before(window.Start) {
			phasePeriodEnd = window.Start
		}
		if !phasePeriodEnd.Before(window.NextStart) {
			phasePeriodEnd = window.NextStart.AddDate(0, 0, -1)
		}
	}

	ovulationDate := window.NextStart.AddDate(0, 0, -defaultOvulationOffset)
	minOvulationDate := phasePeriodEnd.AddDate(0, 0, 1)
	maxOvulationDate := window.NextStart.AddDate(0, 0, -1)
	if ovulationDate.Before(minOvulationDate) {
		ovulationDate = minOvulationDate
	}
	if ovulationDate.After(maxOvulationDate) {
		ovulationDate = maxOvulationDate
	}

	fertileStart := ovulationDate.AddDate(0, 0, -fertileWindowLeadDays)
	if fertileStart.Before(phasePeriodEnd.AddDate(0, 0, 1)) {
		fertileStart = phasePeriodEnd.AddDate(0, 0, 1)
	}
	fertileEnd := ovulationDate

	isRecordedPeriod := false
	if window.ActualCycle != nil {
		recordedEnd := phasePeriodEnd
		if window.ActualCycle.EndDate == nil && day.After(today) {
			recordedEnd = today
		}
		if !recordedEnd.Before(window.Start) && !day.Before(window.Start) && !day.After(recordedEnd) {
			isRecordedPeriod = true
		}
	}

	isPredictedPeriod := false
	if !isRecordedPeriod && !day.Before(window.Start) && !day.After(forecastPeriodEnd) {
		switch {
		case window.StartSource == cycleSourcePredicted:
			isPredictedPeriod = true
		case window.ActualCycle != nil && window.ActualCycle.EndDate == nil && day.After(today):
			isPredictedPeriod = true
		}
	}

	periodStatus := cyclePeriodNone
	phase := cyclePhaseFollicular
	phaseSource := cycleSourcePredicted

	switch {
	case isRecordedPeriod:
		periodStatus = cyclePeriodRecorded
		phase = cyclePhaseMenstrual
		phaseSource = cycleSourceRecorded
	case isPredictedPeriod:
		periodStatus = cyclePeriodPredicted
		phase = cyclePhaseMenstrual
	case sameDay(day, ovulationDate):
		phase = cyclePhaseOvulation
	case day.After(ovulationDate):
		phase = cyclePhaseLuteal
	default:
		phase = cyclePhaseFollicular
	}

	dropletFillRatio, dropletState := dropletStateForDay(
		day,
		phase,
		window.Start,
		phasePeriodEnd,
		ovulationDate,
		window.NextStart,
	)

	daysUntilPhaseChange := 0
	switch phase {
	case cyclePhaseMenstrual:
		daysUntilPhaseChange = maxInt(0, dayDiff(day, phasePeriodEnd.AddDate(0, 0, 1)))
	case cyclePhaseFollicular:
		daysUntilPhaseChange = maxInt(0, dayDiff(day, ovulationDate))
	case cyclePhaseOvulation:
		daysUntilPhaseChange = 1
	case cyclePhaseLuteal:
		daysUntilPhaseChange = maxInt(0, dayDiff(day, window.NextStart))
	}

	return cycleDayState{
		Date:                day,
		Phase:               phase,
		PhaseSource:         phaseSource,
		PeriodStatus:        periodStatus,
		IsFertileWindow:     !day.Before(fertileStart) && !day.After(fertileEnd),
		IsOvulationDay:      sameDay(day, ovulationDate),
		DropletFillRatio:    dropletFillRatio,
		DropletState:        dropletState,
		CycleDayNumber:      maxInt(1, dayDiff(window.Start, day)+1),
		DaysUntilNextPeriod: maxInt(0, dayDiff(day, window.NextStart)),
		DaysUntilPhaseShift: daysUntilPhaseChange,
		CurrentCycleStart:   window.Start,
		CurrentCycleEnd:     currentCycleEnd,
		PredictedPeriodEnd:  forecastPeriodEnd,
		PredictedNextStart:  window.NextStart,
		PredictedOvulation:  ovulationDate,
		FertileWindowStart:  fertileStart,
		FertileWindowEnd:    fertileEnd,
		IsRecordedPeriod:    isRecordedPeriod,
	}
}

func resolveCycleWindow(
	day, selectedDay time.Time,
	cycles []Cycle,
	cycleLen, periodLen int,
) cycleWindow {
	actualStarts := make([]time.Time, 0, len(cycles))
	cycleByStart := make(map[string]Cycle, len(cycles))
	for _, cycle := range cycles {
		start := dateOnly(cycle.StartDate)
		key := start.Format("2006-01-02")
		if _, exists := cycleByStart[key]; !exists {
			cycleByStart[key] = cycle
			actualStarts = append(actualStarts, start)
		}
	}
	sort.Slice(actualStarts, func(i, j int) bool {
		return actualStarts[i].Before(actualStarts[j])
	})

	fallbackStart := dateOnly(selectedDay).AddDate(0, 0, -maxInt(periodLen, 1))
	if len(actualStarts) == 0 {
		currentStart := fallbackStart
		nextStart := currentStart.AddDate(0, 0, cycleLen)
		for day.Before(currentStart) {
			nextStart = currentStart
			currentStart = currentStart.AddDate(0, 0, -cycleLen)
		}
		for !day.Before(nextStart) {
			currentStart = nextStart
			nextStart = currentStart.AddDate(0, 0, cycleLen)
		}
		return cycleWindow{
			Start:       currentStart,
			NextStart:   nextStart,
			StartSource: cycleSourcePredicted,
		}
	}

	index := sort.Search(len(actualStarts), func(i int) bool {
		return !actualStarts[i].Before(day)
	})

	if index == 0 && day.Before(actualStarts[0]) {
		nextStart := actualStarts[0]
		currentStart := nextStart.AddDate(0, 0, -cycleLen)
		for day.Before(currentStart) {
			nextStart = currentStart
			currentStart = currentStart.AddDate(0, 0, -cycleLen)
		}
		return cycleWindow{
			Start:       currentStart,
			NextStart:   nextStart,
			StartSource: cycleSourcePredicted,
		}
	}

	currentStart := actualStarts[maxInt(index-1, 0)]
	currentSource := cycleSourceRecorded
	nextStart := currentStart.AddDate(0, 0, cycleLen)
	if index < len(actualStarts) && actualStarts[index].After(currentStart) {
		nextStart = actualStarts[index]
	}
	if index < len(actualStarts) && sameDay(actualStarts[index], day) {
		currentStart = actualStarts[index]
		currentSource = cycleSourceRecorded
		if index+1 < len(actualStarts) {
			nextStart = actualStarts[index+1]
		} else {
			nextStart = currentStart.AddDate(0, 0, cycleLen)
		}
	}
	for !day.Before(nextStart) {
		currentStart = nextStart
		nextStart = currentStart.AddDate(0, 0, cycleLen)
		currentSource = cycleSourcePredicted
	}

	var actualCycle *Cycle
	if currentSource == cycleSourceRecorded {
		if cycle, ok := cycleByStart[currentStart.Format("2006-01-02")]; ok {
			copyCycle := cycle
			actualCycle = &copyCycle
		}
	}

	return cycleWindow{
		Start:       currentStart,
		NextStart:   nextStart,
		StartSource: currentSource,
		ActualCycle: actualCycle,
	}
}

func recordedCycleEnd(cycle *Cycle, today time.Time) *time.Time {
	if cycle == nil {
		return nil
	}
	start := dateOnly(cycle.StartDate)
	if cycle.EndDate != nil {
		end := dateOnly(*cycle.EndDate)
		if end.Before(start) {
			end = start
		}
		return &end
	}
	if today.Before(start) {
		return nil
	}
	end := today
	return &end
}

func dropletStateForDay(
	day time.Time,
	phase string,
	periodStart, periodEnd, ovulationDate, nextStart time.Time,
) (float64, string) {
	switch phase {
	case cyclePhaseMenstrual:
		totalSteps := maxInt(1, dayDiff(periodStart, periodEnd))
		progress := clampFloat64(float64(dayDiff(periodStart, day))/float64(totalSteps), 0, 1)
		return clampFloat64(1.0-progress*(1.0-0.08), 0.08, 1.0), dropletStateEmptying
	case cyclePhaseLuteal:
		lutealStart := ovulationDate.AddDate(0, 0, 1)
		lutealEnd := nextStart.AddDate(0, 0, -1)
		totalSteps := dayDiff(lutealStart, lutealEnd)
		if totalSteps <= 0 {
			return 1.0, dropletStateFilling
		}
		progress := clampFloat64(float64(dayDiff(lutealStart, day))/float64(totalSteps), 0, 1)
		return clampFloat64(0.18+progress*(1.0-0.18), 0.18, 1.0), dropletStateFilling
	case cyclePhaseOvulation:
		return 0.18, dropletStateSteady
	default:
		return 0.08, dropletStateSteady
	}
}

func effectiveCaloriesTarget(baseCalories float64) float64 {
	if baseCalories <= 0 {
		return baseCalories
	}
	return roundTo10(baseCalories * periodCaloriesMultiplier)
}

func roundTo10(value float64) float64 {
	return math.Round(value/10.0) * 10.0
}

func cycleConfidence(historyCount int) string {
	switch {
	case historyCount >= 6:
		return "high"
	case historyCount >= 3:
		return "medium"
	default:
		return "low"
	}
}

func normalizeCycleSeedMode(value string) (string, error) {
	mode := strings.ToLower(strings.TrimSpace(value))
	switch mode {
	case cycleSeedModeStarted, cycleSeedModeEnded:
		return mode, nil
	default:
		return "", ErrCycleInputInvalid
	}
}

func medianInt(values []int) int {
	if len(values) == 0 {
		return 0
	}
	copyVals := append([]int(nil), values...)
	sort.Ints(copyVals)
	mid := len(copyVals) / 2
	if len(copyVals)%2 == 1 {
		return copyVals[mid]
	}
	return int(math.Round(float64(copyVals[mid-1]+copyVals[mid]) / 2.0))
}

func clampInt(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func minInt(value, other int) int {
	if value < other {
		return value
	}
	return other
}

func maxInt(value, other int) int {
	if value > other {
		return value
	}
	return other
}

func dayDiff(start, end time.Time) int {
	return int(dateOnly(end).Sub(dateOnly(start)).Hours() / 24)
}

func sameDay(left, right time.Time) bool {
	return dateOnly(left).Equal(dateOnly(right))
}

func clampFloat64(value, minValue, maxValue float64) float64 {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func dateOnly(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
