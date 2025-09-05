package body

import (
    "context"
    "math"
    "time"

    "github.com/jmoiron/sqlx"

    "github.com/jourloy/nutri-backend/internal/database"
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
    // plateau history
    GetPlateauHistory(ctx context.Context, userId string, from, to *time.Time) ([]PlateauEvent, error)
}

type service struct {
    repo Repository
    db   *sqlx.DB
}

func NewService() Service { return &service{repo: NewRepository(), db: database.Database} }

// passthrough
func (s *service) CreateWeight(ctx context.Context, w WeightCreate) (*Weight, error) { return s.repo.CreateWeight(ctx, w) }
func (s *service) UpdateWeight(ctx context.Context, w Weight) (*Weight, error) { return s.repo.UpdateWeight(ctx, w) }
func (s *service) DeleteWeight(ctx context.Context, id int64, userId string) error { return s.repo.DeleteWeight(ctx, id, userId) }
func (s *service) GetWeights(ctx context.Context, userId string, from, to *time.Time) ([]Weight, error) {
    return s.repo.GetWeights(ctx, userId, from, to)
}
func (s *service) GetLatestWeight(ctx context.Context, userId string) (*Weight, error) { return s.repo.GetLatestWeight(ctx, userId) }
func (s *service) CreateMeasurement(ctx context.Context, m MeasurementCreate) (*Measurement, error) { return s.repo.CreateMeasurement(ctx, m) }
func (s *service) UpdateMeasurement(ctx context.Context, m Measurement) (*Measurement, error) { return s.repo.UpdateMeasurement(ctx, m) }
func (s *service) DeleteMeasurement(ctx context.Context, id int64, userId string) error { return s.repo.DeleteMeasurement(ctx, id, userId) }
func (s *service) GetMeasurements(ctx context.Context, userId string, from, to *time.Time) ([]Measurement, error) {
    return s.repo.GetMeasurements(ctx, userId, from, to)
}
func (s *service) GetLatestMeasurement(ctx context.Context, userId string) (*Measurement, error) {
    return s.repo.GetLatestMeasurement(ctx, userId)
}

// activity passthrough
func (s *service) CreateActivity(ctx context.Context, a ActivityCreate) (*Activity, error) { return s.repo.CreateActivity(ctx, a) }
func (s *service) UpdateActivity(ctx context.Context, a Activity) (*Activity, error) { return s.repo.UpdateActivity(ctx, a) }
func (s *service) DeleteActivity(ctx context.Context, id int64, userId string) error { return s.repo.DeleteActivity(ctx, id, userId) }
func (s *service) GetActivity(ctx context.Context, userId string, from, to *time.Time) ([]Activity, error) {
    return s.repo.GetActivity(ctx, userId, from, to)
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
    if err != nil { return nil, err }
    // build series by day ascending
    type dp struct{ d time.Time; v float64 }
    m := map[string]float64{}
    for _, w := range ws { m[w.LoggedAt.Format("2006-01-02")] = w.Value }
    series := make([]dp, 0, windowDays)
    for d := 0; d < windowDays; d++ {
        day := start.AddDate(0,0,d)
        if v, ok := m[day.Format("2006-01-02")]; ok {
            series = append(series, dp{d: day, v: v})
        }
    }
    if len(series) < 7 {
        // not enough data
        res := &PlateauResult{IsPlateau: false, Goal: "unknown", WindowStart: start.Format("2006-01-02"), WindowEnd: end.Format("2006-01-02"), WindowDays: windowDays, DaysWithWeight: len(series), Reason: "Недостаточно данных (минимум 7 дней)"}
        return res, nil
    }

    // EWMA smoothing (lambda=0.2)
    lambda := 0.2
    sm := make([]float64, len(series))
    sm[0] = series[0].v
    for i := 1; i < len(series); i++ { sm[i] = lambda*series[i].v + (1-lambda)*sm[i-1] }

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
    if denom == 0 { denom = 1 }
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
    if goal == "" { goal = "unknown" }

    // Compliance: calories +/- 10%, protein >= 1.6g/kg
    dailyCals, _ := s.repo.GetDailyCalories(ctx, userId, start, end)
    dailyProt, _ := s.repo.GetDailyProtein(ctx, userId, start, end)
    dailySteps, _ := s.repo.GetDailySteps(ctx, userId, start, end)
    dailySleep, _ := s.repo.GetDailySleepMin(ctx, userId, start, end)
    calsGood := 0
    protGood := 0
    lower := 0.9 * targetCalories
    upper := 1.1 * targetCalories
    proteinTarget := 1.6 * profileWeight
    // iterate days in window
    for d := 0; d < windowDays; d++ {
        day := start.AddDate(0,0,d).Format("2006-01-02")
        if v, ok := dailyCals[day]; ok && targetCalories > 0 && v >= lower && v <= upper { calsGood++ }
        if v, ok := dailyProt[day]; ok && proteinTarget > 0 && v >= proteinTarget { protGood++ }
    }
    // steps & sleep averages across the window
    var stepsSum int
    var sleepMinSum int
    for d := 0; d < windowDays; d++ {
        day := start.AddDate(0,0,d).Format("2006-01-02")
        if v, ok := dailySteps[day]; ok { stepsSum += v }
        if v, ok := dailySleep[day]; ok { sleepMinSum += v }
    }
    stepsAvg := float64(stepsSum) / float64(windowDays)
    sleepAvgHours := float64(sleepMinSum) / float64(windowDays*60)
    // steps target from fit profile
    var stepsTarget int
    _ = s.db.GetContext(ctx, &stepsTarget, `SELECT steps_target FROM fit_profiles WHERE user_id=$1 ORDER BY created_at DESC LIMIT 1`, userId)
    if stepsTarget <= 0 { stepsTarget = 8000 }

    sufficient := (calsGood >= 14) && (protGood >= 14) && (stepsAvg >= 0.8*float64(stepsTarget)) && (sleepAvgHours >= 6.0)

    // Plateau rules
    isPlateau := false
    reason := ""
    // base thresholds
    thrPct := 0.05 // per week
    smallDelta := 0.3 // kg

    switch goal {
    case "lose", "fat_loss", "weight_loss":
        if (slopeWeeklyPct > -thrPct) || (math.Abs(deltaKg) < smallDelta) { isPlateau = true; reason = "Слабый тренд снижения веса или малая динамика" }
    case "gain", "muscle_gain", "bulk":
        if (slopeWeeklyPct < thrPct) || (math.Abs(deltaKg) < smallDelta) { isPlateau = true; reason = "Слабый тренд набора веса или малая динамика" }
    default:
        // if unknown goal — decide only by flat trend
        if math.Abs(slopeWeeklyPct) < thrPct && math.Abs(deltaKg) < smallDelta { isPlateau = true; reason = "Плоский тренд веса" }
    }

    if isPlateau && !sufficient {
        // compliance not sufficient -> do not flag plateau strictly
        isPlateau = false
        if reason == "" { reason = "Недостаточное соблюдение режима" } else { reason += "; недостаточное соблюдение" }
    }

    // persist event
    _, _ = s.db.ExecContext(ctx, `
        INSERT INTO body_plateau_events (user_id, window_start, window_end, goal, slope_weekly_pct, delta_kg, days_with_weight, calories_good_days, protein_good_days, window_days, is_plateau, reason)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
        userId, start, end, goal, slopeWeeklyPct, deltaKg, len(series), calsGood, protGood, windowDays, isPlateau, reason,
    )

    res := &PlateauResult{
        IsPlateau: isPlateau,
        Goal: goal,
        WindowStart: start.Format("2006-01-02"),
        WindowEnd: end.Format("2006-01-02"),
        WindowDays: windowDays,
        DaysWithWeight: len(series),
        SlopeWeeklyPct: slopeWeeklyPct,
        DeltaKg: deltaKg,
        CaloriesGoodDays: calsGood,
        ProteinGoodDays: protGood,
        CaloriesTarget: targetCalories,
        ProteinPerKg: 1.6,
        StepsAvg: stepsAvg,
        StepsTarget: stepsTarget,
        SleepAvgHours: sleepAvgHours,
        Reason: reason,
    }
    return res, nil
}
