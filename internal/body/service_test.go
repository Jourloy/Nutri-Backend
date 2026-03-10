package body

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type stubRepo struct {
	getLatestGenderFn       func(ctx context.Context, userId string) (string, error)
	getLatestMacroGoalsFn   func(ctx context.Context, userId string) (CycleGoals, error)
	getPeriodDaysFn         func(ctx context.Context, userId string, from, to time.Time) (map[string]bool, error)
	getCyclesFn             func(ctx context.Context, userId string, from, to *time.Time, limit int) ([]Cycle, error)
	getOpenCycleFn          func(ctx context.Context, userId string) (*Cycle, error)
	createWorkoutFn         func(ctx context.Context, w WorkoutCreate) (*Workout, error)
	updateWorkoutFn         func(ctx context.Context, w WorkoutCreate, id int64) (*Workout, error)
	deleteWorkoutFn         func(ctx context.Context, id int64, userId string) error
	getWorkoutsFn           func(ctx context.Context, userId string, from, to *time.Time) ([]Workout, error)
	getWorkoutTotalByDateFn func(ctx context.Context, userId string, date time.Time) (int, error)
	getLatestWaterLimitFn   func(ctx context.Context, userId string) (int, error)
	upsertCycleDayLogFn     func(ctx context.Context, userId string, loggedAt time.Time, flowIntensity, note *string) (int64, error)
	replaceCycleDayEventsFn func(ctx context.Context, dayLogId int64, events []CycleDayEventInput) error
	getCycleDayLogsFn       func(ctx context.Context, userId string, from, to *time.Time) ([]CycleDayLog, error)
	startCycleFn            func(ctx context.Context, userId string, startDate time.Time) error
	stopCycleFn             func(ctx context.Context, userId string, endDate time.Time) (*Cycle, error)
}

func (s stubRepo) CreateWeight(ctx context.Context, w WeightCreate) (*Weight, error) {
	return nil, nil
}
func (s stubRepo) UpdateWeight(ctx context.Context, w Weight) (*Weight, error) {
	return nil, nil
}
func (s stubRepo) DeleteWeight(ctx context.Context, id int64, userId string) error {
	return nil
}
func (s stubRepo) GetWeights(ctx context.Context, userId string, from, to *time.Time) ([]Weight, error) {
	return nil, nil
}
func (s stubRepo) GetLatestWeight(ctx context.Context, userId string) (*Weight, error) {
	return nil, nil
}
func (s stubRepo) CreateMeasurement(ctx context.Context, m MeasurementCreate) (*Measurement, error) {
	return nil, nil
}
func (s stubRepo) UpdateMeasurement(ctx context.Context, m Measurement) (*Measurement, error) {
	return nil, nil
}
func (s stubRepo) DeleteMeasurement(ctx context.Context, id int64, userId string) error {
	return nil
}
func (s stubRepo) GetMeasurements(ctx context.Context, userId string, from, to *time.Time) ([]Measurement, error) {
	return nil, nil
}
func (s stubRepo) GetLatestMeasurement(ctx context.Context, userId string) (*Measurement, error) {
	return nil, nil
}
func (s stubRepo) GetDailyCalories(ctx context.Context, userId string, from, to time.Time) (map[string]float64, error) {
	return nil, nil
}
func (s stubRepo) GetDailyProtein(ctx context.Context, userId string, from, to time.Time) (map[string]float64, error) {
	return nil, nil
}
func (s stubRepo) GetDailySteps(ctx context.Context, userId string, from, to time.Time) (map[string]int, error) {
	return nil, nil
}
func (s stubRepo) GetDailySleepMin(ctx context.Context, userId string, from, to time.Time) (map[string]int, error) {
	return nil, nil
}
func (s stubRepo) GetPeriodDays(ctx context.Context, userId string, from, to time.Time) (map[string]bool, error) {
	if s.getPeriodDaysFn != nil {
		return s.getPeriodDaysFn(ctx, userId, from, to)
	}
	return map[string]bool{}, nil
}
func (s stubRepo) GetLatestGender(ctx context.Context, userId string) (string, error) {
	if s.getLatestGenderFn != nil {
		return s.getLatestGenderFn(ctx, userId)
	}
	return "female", nil
}
func (s stubRepo) GetLatestMacroGoals(ctx context.Context, userId string) (CycleGoals, error) {
	if s.getLatestMacroGoalsFn != nil {
		return s.getLatestMacroGoalsFn(ctx, userId)
	}
	return CycleGoals{}, nil
}
func (s stubRepo) CreateActivity(ctx context.Context, a ActivityCreate) (*Activity, error) {
	return nil, nil
}
func (s stubRepo) UpdateActivity(ctx context.Context, a Activity) (*Activity, error) {
	return nil, nil
}
func (s stubRepo) DeleteActivity(ctx context.Context, id int64, userId string) error {
	return nil
}
func (s stubRepo) GetActivity(ctx context.Context, userId string, from, to *time.Time) ([]Activity, error) {
	return nil, nil
}
func (s stubRepo) CreateWorkout(ctx context.Context, w WorkoutCreate) (*Workout, error) {
	if s.createWorkoutFn != nil {
		return s.createWorkoutFn(ctx, w)
	}
	return &Workout{
		Id:          1,
		LoggedAt:    w.LoggedAt.Format("2006-01-02"),
		DurationMin: w.DurationMin,
		WorkoutType: w.WorkoutType,
		Intensity:   w.Intensity,
		Note:        w.Note,
	}, nil
}
func (s stubRepo) UpdateWorkout(ctx context.Context, w WorkoutCreate, id int64) (*Workout, error) {
	if s.updateWorkoutFn != nil {
		return s.updateWorkoutFn(ctx, w, id)
	}
	return &Workout{
		Id:          id,
		LoggedAt:    w.LoggedAt.Format("2006-01-02"),
		DurationMin: w.DurationMin,
		WorkoutType: w.WorkoutType,
		Intensity:   w.Intensity,
		Note:        w.Note,
	}, nil
}
func (s stubRepo) DeleteWorkout(ctx context.Context, id int64, userId string) error {
	if s.deleteWorkoutFn != nil {
		return s.deleteWorkoutFn(ctx, id, userId)
	}
	return nil
}
func (s stubRepo) GetWorkouts(ctx context.Context, userId string, from, to *time.Time) ([]Workout, error) {
	if s.getWorkoutsFn != nil {
		return s.getWorkoutsFn(ctx, userId, from, to)
	}
	return nil, nil
}
func (s stubRepo) GetWorkoutTotalDurationByDate(ctx context.Context, userId string, date time.Time) (int, error) {
	if s.getWorkoutTotalByDateFn != nil {
		return s.getWorkoutTotalByDateFn(ctx, userId, date)
	}
	return 0, nil
}
func (s stubRepo) GetLatestWaterLimit(ctx context.Context, userId string) (int, error) {
	if s.getLatestWaterLimitFn != nil {
		return s.getLatestWaterLimitFn(ctx, userId)
	}
	return defaultWaterLimitMl, nil
}
func (s stubRepo) GetPlateauHistory(ctx context.Context, userId string, from, to *time.Time) ([]PlateauEvent, error) {
	return nil, nil
}
func (s stubRepo) StartCycle(ctx context.Context, userId string, startDate time.Time) error {
	if s.startCycleFn != nil {
		return s.startCycleFn(ctx, userId, startDate)
	}
	return nil
}
func (s stubRepo) StopCycle(ctx context.Context, userId string, endDate time.Time) (*Cycle, error) {
	if s.stopCycleFn != nil {
		return s.stopCycleFn(ctx, userId, endDate)
	}
	return nil, nil
}
func (s stubRepo) GetOpenCycle(ctx context.Context, userId string) (*Cycle, error) {
	if s.getOpenCycleFn != nil {
		return s.getOpenCycleFn(ctx, userId)
	}
	return nil, nil
}
func (s stubRepo) GetCycles(ctx context.Context, userId string, from, to *time.Time, limit int) ([]Cycle, error) {
	if s.getCyclesFn != nil {
		return s.getCyclesFn(ctx, userId, from, to, limit)
	}
	return nil, nil
}
func (s stubRepo) UpsertCycleDayLog(
	ctx context.Context,
	userId string,
	loggedAt time.Time,
	flowIntensity, note *string,
) (int64, error) {
	if s.upsertCycleDayLogFn != nil {
		return s.upsertCycleDayLogFn(ctx, userId, loggedAt, flowIntensity, note)
	}
	return 1, nil
}
func (s stubRepo) ReplaceCycleDayEvents(ctx context.Context, dayLogId int64, events []CycleDayEventInput) error {
	if s.replaceCycleDayEventsFn != nil {
		return s.replaceCycleDayEventsFn(ctx, dayLogId, events)
	}
	return nil
}
func (s stubRepo) GetCycleDayLogs(ctx context.Context, userId string, from, to *time.Time) ([]CycleDayLog, error) {
	if s.getCycleDayLogsFn != nil {
		return s.getCycleDayLogsFn(ctx, userId, from, to)
	}
	return nil, nil
}

func mustDate(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		t.Fatalf("parse date: %v", err)
	}
	return parsed
}

func TestGetCycleSummary_DefaultForecast(t *testing.T) {
	svc := &service{
		repo: stubRepo{
			getLatestGenderFn: func(ctx context.Context, userId string) (string, error) {
				return "female", nil
			},
			getLatestMacroGoalsFn: func(ctx context.Context, userId string) (CycleGoals, error) {
				return CycleGoals{Calories: 1800, Protein: 120, Fat: 60, Carbs: 180}, nil
			},
			getPeriodDaysFn: func(ctx context.Context, userId string, from, to time.Time) (map[string]bool, error) {
				return map[string]bool{}, nil
			},
			getCyclesFn: func(ctx context.Context, userId string, from, to *time.Time, limit int) ([]Cycle, error) {
				return []Cycle{}, nil
			},
		},
	}

	at := mustDate(t, "2026-02-18")
	summary, err := svc.GetCycleSummary(context.Background(), "u1", &at)
	if err != nil {
		t.Fatalf("GetCycleSummary returned error: %v", err)
	}

	if summary.CycleLengthDays != 28 {
		t.Fatalf("expected default cycle length 28, got %d", summary.CycleLengthDays)
	}
	if summary.PeriodLengthDays != 5 {
		t.Fatalf("expected default period length 5, got %d", summary.PeriodLengthDays)
	}
	if summary.Confidence != "low" {
		t.Fatalf("expected low confidence, got %s", summary.Confidence)
	}
}

func TestGetCycleSummary_MedianForecast(t *testing.T) {
	cycles := []Cycle{
		{StartDate: mustDate(t, "2025-05-24"), EndDate: ptrDate(mustDate(t, "2025-05-29"))},
		{StartDate: mustDate(t, "2025-04-27"), EndDate: ptrDate(mustDate(t, "2025-05-02"))},
		{StartDate: mustDate(t, "2025-03-30"), EndDate: ptrDate(mustDate(t, "2025-04-03"))},
		{StartDate: mustDate(t, "2025-03-02"), EndDate: ptrDate(mustDate(t, "2025-03-07"))},
	}

	svc := &service{
		repo: stubRepo{
			getLatestGenderFn: func(ctx context.Context, userId string) (string, error) {
				return "female", nil
			},
			getLatestMacroGoalsFn: func(ctx context.Context, userId string) (CycleGoals, error) {
				return CycleGoals{Calories: 1700, Protein: 100, Fat: 50, Carbs: 200}, nil
			},
			getPeriodDaysFn: func(ctx context.Context, userId string, from, to time.Time) (map[string]bool, error) {
				return map[string]bool{}, nil
			},
			getCyclesFn: func(ctx context.Context, userId string, from, to *time.Time, limit int) ([]Cycle, error) {
				return cycles, nil
			},
		},
	}

	at := mustDate(t, "2025-05-25")
	summary, err := svc.GetCycleSummary(context.Background(), "u1", &at)
	if err != nil {
		t.Fatalf("GetCycleSummary returned error: %v", err)
	}

	if summary.CycleLengthDays != 28 {
		t.Fatalf("expected median cycle length 28, got %d", summary.CycleLengthDays)
	}
	if summary.PeriodLengthDays != 6 {
		t.Fatalf("expected median period length 6, got %d", summary.PeriodLengthDays)
	}
	if summary.Confidence != "medium" {
		t.Fatalf("expected medium confidence, got %s", summary.Confidence)
	}
}

func TestGetCycleTimeline_DefaultForecast(t *testing.T) {
	svc := &service{
		repo: stubRepo{
			getLatestGenderFn: func(ctx context.Context, userId string) (string, error) {
				return "female", nil
			},
			getLatestMacroGoalsFn: func(ctx context.Context, userId string) (CycleGoals, error) {
				return CycleGoals{Calories: 1800, Protein: 120, Fat: 60, Carbs: 180}, nil
			},
			getCyclesFn: func(ctx context.Context, userId string, from, to *time.Time, limit int) ([]Cycle, error) {
				return []Cycle{}, nil
			},
			getCycleDayLogsFn: func(ctx context.Context, userId string, from, to *time.Time) ([]CycleDayLog, error) {
				return nil, nil
			},
		},
	}

	from := mustDate(t, "2026-02-16")
	to := mustDate(t, "2026-02-22")
	at := mustDate(t, "2026-02-18")
	timeline, err := svc.GetCycleTimeline(context.Background(), "u1", &from, &to, &at)
	if err != nil {
		t.Fatalf("GetCycleTimeline returned error: %v", err)
	}

	if timeline.Summary.CycleLengthDays != 28 {
		t.Fatalf("expected default cycle length 28, got %d", timeline.Summary.CycleLengthDays)
	}
	if timeline.Summary.PeriodLengthDays != 5 {
		t.Fatalf("expected default period length 5, got %d", timeline.Summary.PeriodLengthDays)
	}
	if timeline.Summary.Phase != cyclePhaseFollicular {
		t.Fatalf("expected follicular phase, got %s", timeline.Summary.Phase)
	}
	if timeline.Summary.PeriodStatus != cyclePeriodNone {
		t.Fatalf("expected no period status, got %s", timeline.Summary.PeriodStatus)
	}
	if timeline.Summary.CycleDayNumber != 6 {
		t.Fatalf("expected cycle day 6, got %d", timeline.Summary.CycleDayNumber)
	}
	if timeline.Summary.DaysUntilNextPeriod != 23 {
		t.Fatalf("expected 23 days until next period, got %d", timeline.Summary.DaysUntilNextPeriod)
	}
	if timeline.Summary.DropletFillRatio != 0.08 {
		t.Fatalf("expected droplet fill 0.08, got %.2f", timeline.Summary.DropletFillRatio)
	}
}

func TestGetCycleTimeline_MedianForecast(t *testing.T) {
	cycles := []Cycle{
		{StartDate: mustDate(t, "2025-05-24"), EndDate: ptrDate(mustDate(t, "2025-05-29"))},
		{StartDate: mustDate(t, "2025-04-27"), EndDate: ptrDate(mustDate(t, "2025-05-02"))},
		{StartDate: mustDate(t, "2025-03-30"), EndDate: ptrDate(mustDate(t, "2025-04-03"))},
		{StartDate: mustDate(t, "2025-03-02"), EndDate: ptrDate(mustDate(t, "2025-03-07"))},
	}

	svc := &service{
		repo: stubRepo{
			getLatestGenderFn: func(ctx context.Context, userId string) (string, error) {
				return "female", nil
			},
			getLatestMacroGoalsFn: func(ctx context.Context, userId string) (CycleGoals, error) {
				return CycleGoals{Calories: 1700, Protein: 100, Fat: 50, Carbs: 200}, nil
			},
			getCyclesFn: func(ctx context.Context, userId string, from, to *time.Time, limit int) ([]Cycle, error) {
				return cycles, nil
			},
			getCycleDayLogsFn: func(ctx context.Context, userId string, from, to *time.Time) ([]CycleDayLog, error) {
				return nil, nil
			},
		},
	}

	from := mustDate(t, "2025-05-24")
	to := mustDate(t, "2025-06-04")
	at := mustDate(t, "2025-05-25")
	timeline, err := svc.GetCycleTimeline(context.Background(), "u1", &from, &to, &at)
	if err != nil {
		t.Fatalf("GetCycleTimeline returned error: %v", err)
	}

	if timeline.Summary.Phase != cyclePhaseMenstrual {
		t.Fatalf("expected menstrual phase, got %s", timeline.Summary.Phase)
	}
	if timeline.Summary.PeriodStatus != cyclePeriodRecorded {
		t.Fatalf("expected recorded period, got %s", timeline.Summary.PeriodStatus)
	}
	if timeline.Summary.CurrentCycleStart == nil || *timeline.Summary.CurrentCycleStart != "2025-05-24" {
		t.Fatalf("unexpected currentCycleStart: %v", timeline.Summary.CurrentCycleStart)
	}
	if timeline.Summary.CurrentCycleEnd == nil || *timeline.Summary.CurrentCycleEnd != "2025-05-29" {
		t.Fatalf("unexpected currentCycleEnd: %v", timeline.Summary.CurrentCycleEnd)
	}
	if timeline.Summary.PredictedNextStart == nil || *timeline.Summary.PredictedNextStart != "2025-06-21" {
		t.Fatalf("unexpected predictedNextStart: %v", timeline.Summary.PredictedNextStart)
	}
	if timeline.Summary.PredictedOvulationDate == nil || *timeline.Summary.PredictedOvulationDate != "2025-06-07" {
		t.Fatalf("unexpected ovulation date: %v", timeline.Summary.PredictedOvulationDate)
	}
}

func TestGetCycleTimeline_RecordedPeriodOverridesForecast(t *testing.T) {
	cycles := []Cycle{
		{StartDate: mustDate(t, "2025-05-24"), EndDate: ptrDate(mustDate(t, "2025-05-25"))},
		{StartDate: mustDate(t, "2025-04-27"), EndDate: ptrDate(mustDate(t, "2025-05-02"))},
		{StartDate: mustDate(t, "2025-03-30"), EndDate: ptrDate(mustDate(t, "2025-04-03"))},
		{StartDate: mustDate(t, "2025-03-02"), EndDate: ptrDate(mustDate(t, "2025-03-07"))},
	}

	svc := &service{
		repo: stubRepo{
			getLatestGenderFn: func(ctx context.Context, userId string) (string, error) {
				return "female", nil
			},
			getLatestMacroGoalsFn: func(ctx context.Context, userId string) (CycleGoals, error) {
				return CycleGoals{Calories: 1800, Protein: 120, Fat: 60, Carbs: 180}, nil
			},
			getCyclesFn: func(ctx context.Context, userId string, from, to *time.Time, limit int) ([]Cycle, error) {
				return cycles, nil
			},
			getCycleDayLogsFn: func(ctx context.Context, userId string, from, to *time.Time) ([]CycleDayLog, error) {
				return nil, nil
			},
		},
	}

	from := mustDate(t, "2025-05-26")
	to := mustDate(t, "2025-05-28")
	at := mustDate(t, "2025-05-27")
	timeline, err := svc.GetCycleTimeline(context.Background(), "u1", &from, &to, &at)
	if err != nil {
		t.Fatalf("GetCycleTimeline returned error: %v", err)
	}

	if timeline.Summary.Phase != cyclePhaseFollicular {
		t.Fatalf("expected follicular phase after recorded stop, got %s", timeline.Summary.Phase)
	}
	if timeline.Summary.PeriodStatus != cyclePeriodNone {
		t.Fatalf("expected no period status after recorded stop, got %s", timeline.Summary.PeriodStatus)
	}
}

func TestGetCycleTimeline_FutureRangeRepeatsPredictedCycles(t *testing.T) {
	cycles := []Cycle{
		{StartDate: mustDate(t, "2025-05-24"), EndDate: ptrDate(mustDate(t, "2025-05-29"))},
		{StartDate: mustDate(t, "2025-04-27"), EndDate: ptrDate(mustDate(t, "2025-05-02"))},
		{StartDate: mustDate(t, "2025-03-30"), EndDate: ptrDate(mustDate(t, "2025-04-03"))},
		{StartDate: mustDate(t, "2025-03-02"), EndDate: ptrDate(mustDate(t, "2025-03-07"))},
	}

	svc := &service{
		repo: stubRepo{
			getLatestGenderFn: func(ctx context.Context, userId string) (string, error) {
				return "female", nil
			},
			getLatestMacroGoalsFn: func(ctx context.Context, userId string) (CycleGoals, error) {
				return CycleGoals{Calories: 1700, Protein: 100, Fat: 50, Carbs: 200}, nil
			},
			getCyclesFn: func(ctx context.Context, userId string, from, to *time.Time, limit int) ([]Cycle, error) {
				return cycles, nil
			},
			getCycleDayLogsFn: func(ctx context.Context, userId string, from, to *time.Time) ([]CycleDayLog, error) {
				return nil, nil
			},
		},
	}

	from := mustDate(t, "2025-06-20")
	to := mustDate(t, "2025-07-25")
	at := mustDate(t, "2025-06-22")
	timeline, err := svc.GetCycleTimeline(context.Background(), "u1", &from, &to, &at)
	if err != nil {
		t.Fatalf("GetCycleTimeline returned error: %v", err)
	}

	if findTimelineDay(t, timeline, "2025-06-21").PeriodStatus != cyclePeriodPredicted {
		t.Fatalf("expected 2025-06-21 to be a predicted period day")
	}
	if findTimelineDay(t, timeline, "2025-07-19").PeriodStatus != cyclePeriodPredicted {
		t.Fatalf("expected 2025-07-19 to be a repeated predicted period day")
	}
	if !findTimelineDay(t, timeline, "2025-07-05").IsOvulationDay {
		t.Fatalf("expected 2025-07-05 to be marked as ovulation day")
	}
}

func TestGetCycleTimeline_UsesDayLogsMarkers(t *testing.T) {
	svc := &service{
		repo: stubRepo{
			getLatestGenderFn: func(ctx context.Context, userId string) (string, error) {
				return "female", nil
			},
			getLatestMacroGoalsFn: func(ctx context.Context, userId string) (CycleGoals, error) {
				return CycleGoals{Calories: 1800, Protein: 120, Fat: 60, Carbs: 180}, nil
			},
			getCyclesFn: func(ctx context.Context, userId string, from, to *time.Time, limit int) ([]Cycle, error) {
				return nil, nil
			},
			getCycleDayLogsFn: func(ctx context.Context, userId string, from, to *time.Time) ([]CycleDayLog, error) {
				return []CycleDayLog{
					{
						Id:            1,
						LoggedAt:      "2026-02-18",
						FlowIntensity: ptrString("medium"),
						Events:        []CycleDailyEvent{},
					},
				}, nil
			},
		},
	}

	from := mustDate(t, "2026-02-16")
	to := mustDate(t, "2026-02-22")
	at := mustDate(t, "2026-02-18")
	timeline, err := svc.GetCycleTimeline(context.Background(), "u1", &from, &to, &at)
	if err != nil {
		t.Fatalf("GetCycleTimeline returned error: %v", err)
	}

	day := findTimelineDay(t, timeline, "2026-02-18")
	if !day.HasLog {
		t.Fatalf("expected log marker on 2026-02-18")
	}
	if day.FlowIntensity == nil || *day.FlowIntensity != "medium" {
		t.Fatalf("expected medium flow marker, got %v", day.FlowIntensity)
	}
}

func TestGetCycleCatalog_NonFemaleForbidden(t *testing.T) {
	svc := &service{
		repo: stubRepo{
			getLatestGenderFn: func(ctx context.Context, userId string) (string, error) {
				return "male", nil
			},
		},
	}

	_, err := svc.GetCycleCatalog(context.Background(), "u1")
	if !errors.Is(err, ErrCycleForbidden) {
		t.Fatalf("expected ErrCycleForbidden, got %v", err)
	}
}

func TestStopCycle_NoActiveCycle(t *testing.T) {
	svc := &service{
		repo: stubRepo{
			getLatestGenderFn: func(ctx context.Context, userId string) (string, error) {
				return "female", nil
			},
			getOpenCycleFn: func(ctx context.Context, userId string) (*Cycle, error) {
				return nil, nil
			},
		},
	}

	_, err := svc.StopCycle(context.Background(), "u1", mustDate(t, "2026-02-18"))
	if !errors.Is(err, ErrCycleNoActive) {
		t.Fatalf("expected ErrCycleNoActive, got %v", err)
	}
}

func TestUpsertCycleDay_ReplacesEvents(t *testing.T) {
	var capturedEvents []CycleDayEventInput
	var capturedLogDate time.Time

	svc := &service{
		repo: stubRepo{
			getLatestGenderFn: func(ctx context.Context, userId string) (string, error) {
				return "female", nil
			},
			upsertCycleDayLogFn: func(ctx context.Context, userId string, loggedAt time.Time, flowIntensity, note *string) (int64, error) {
				capturedLogDate = loggedAt
				return 42, nil
			},
			replaceCycleDayEventsFn: func(ctx context.Context, dayLogId int64, events []CycleDayEventInput) error {
				capturedEvents = events
				if dayLogId != 42 {
					t.Fatalf("expected day log id 42, got %d", dayLogId)
				}
				return nil
			},
			getCycleDayLogsFn: func(ctx context.Context, userId string, from, to *time.Time) ([]CycleDayLog, error) {
				return []CycleDayLog{{Id: 42, LoggedAt: from.Format("2006-01-02"), Events: []CycleDailyEvent{}}}, nil
			},
		},
	}

	flow := "medium"
	note := "note"
	_, err := svc.UpsertCycleDay(context.Background(), "u1", CycleDayUpsertInput{
		LoggedAt:      mustDate(t, "2026-02-18"),
		FlowIntensity: &flow,
		Note:          &note,
		Events: []CycleDayEventInput{
			{EventCategory: "sex", EventCode: "sex_protected", Quantity: 2},
			{EventCategory: "symptoms", EventCode: "cramps", Quantity: 1, Intensity: ptrString("high")},
		},
	})
	if err != nil {
		t.Fatalf("UpsertCycleDay returned error: %v", err)
	}

	if capturedLogDate.Format("2006-01-02") != "2026-02-18" {
		t.Fatalf("unexpected loggedAt: %s", capturedLogDate.Format("2006-01-02"))
	}
	if len(capturedEvents) != 2 {
		t.Fatalf("expected 2 events, got %d", len(capturedEvents))
	}
	if capturedEvents[0].Quantity != 2 {
		t.Fatalf("expected quantity 2, got %d", capturedEvents[0].Quantity)
	}
	if capturedEvents[1].Intensity == nil || *capturedEvents[1].Intensity != "high" {
		t.Fatalf("expected intensity high")
	}
}

func TestEffectiveCaloriesTarget_PeriodAdjustment(t *testing.T) {
	if got := effectiveCaloriesTarget(1800); got != 1940 {
		t.Fatalf("expected 1940 for 1800 base, got %.0f", got)
	}
	if got := effectiveCaloriesTarget(1600); got != 1730 {
		t.Fatalf("expected 1730 for 1600 base, got %.0f", got)
	}
	if got := effectiveCaloriesTarget(0); got != 0 {
		t.Fatalf("expected 0 for zero base, got %.0f", got)
	}
}

func TestGetWorkoutDailySummary_AggregatedDurationBonus(t *testing.T) {
	svc := &service{
		repo: stubRepo{
			getLatestWaterLimitFn: func(ctx context.Context, userId string) (int, error) {
				return 2400, nil
			},
			getWorkoutTotalByDateFn: func(ctx context.Context, userId string, date time.Time) (int, error) {
				// Simulate multiple workouts summed by repository.
				return 95, nil
			},
		},
	}

	today := mustDate(t, "2026-02-21")
	summary, err := svc.GetWorkoutDailySummary(context.Background(), "u1", today, today)
	if err != nil {
		t.Fatalf("GetWorkoutDailySummary returned error: %v", err)
	}

	if !summary.Applied {
		t.Fatalf("expected applied=true")
	}
	if summary.WaterBonusMl != 950 {
		t.Fatalf("expected bonus 950, got %d", summary.WaterBonusMl)
	}
	if summary.EffectiveWaterLimitMl != 3350 {
		t.Fatalf("expected effective 3350, got %d", summary.EffectiveWaterLimitMl)
	}
}

func TestGetWorkoutDailySummary_ClampByMax(t *testing.T) {
	svc := &service{
		repo: stubRepo{
			getLatestWaterLimitFn: func(ctx context.Context, userId string) (int, error) {
				return 4300, nil
			},
			getWorkoutTotalByDateFn: func(ctx context.Context, userId string, date time.Time) (int, error) {
				return 60, nil
			},
		},
	}

	today := mustDate(t, "2026-02-21")
	summary, err := svc.GetWorkoutDailySummary(context.Background(), "u1", today, today)
	if err != nil {
		t.Fatalf("GetWorkoutDailySummary returned error: %v", err)
	}

	if summary.EffectiveWaterLimitMl != 4500 {
		t.Fatalf("expected effective 4500, got %d", summary.EffectiveWaterLimitMl)
	}
	if summary.MaxWaterLimitMl != 4500 {
		t.Fatalf("expected max 4500, got %d", summary.MaxWaterLimitMl)
	}
}

func TestGetWorkoutDailySummary_NoBonusForNonToday(t *testing.T) {
	svc := &service{
		repo: stubRepo{
			getLatestWaterLimitFn: func(ctx context.Context, userId string) (int, error) {
				return 2600, nil
			},
			getWorkoutTotalByDateFn: func(ctx context.Context, userId string, date time.Time) (int, error) {
				return 120, nil
			},
		},
	}

	date := mustDate(t, "2026-02-19")
	today := mustDate(t, "2026-02-21")
	summary, err := svc.GetWorkoutDailySummary(context.Background(), "u1", date, today)
	if err != nil {
		t.Fatalf("GetWorkoutDailySummary returned error: %v", err)
	}

	if summary.Applied {
		t.Fatalf("expected applied=false")
	}
	if summary.WaterBonusMl != 0 {
		t.Fatalf("expected bonus 0, got %d", summary.WaterBonusMl)
	}
	if summary.EffectiveWaterLimitMl != 2600 {
		t.Fatalf("expected effective 2600, got %d", summary.EffectiveWaterLimitMl)
	}
}

func TestCreateWorkout_RejectsFutureDate(t *testing.T) {
	svc := &service{repo: stubRepo{}}

	today := mustDate(t, "2026-02-21")
	future := mustDate(t, "2026-02-22")
	_, err := svc.CreateWorkout(context.Background(), WorkoutCreate{
		UserId:      "u1",
		LoggedAt:    future,
		DurationMin: 45,
		WorkoutType: "cardio",
	}, today)

	if !errors.Is(err, ErrWorkoutFutureDate) {
		t.Fatalf("expected ErrWorkoutFutureDate, got %v", err)
	}
}

func TestCreateWorkout_ValidatesEnumsAndRanges(t *testing.T) {
	svc := &service{repo: stubRepo{}}
	today := mustDate(t, "2026-02-21")

	cases := []WorkoutCreate{
		{UserId: "u1", LoggedAt: today, DurationMin: 0, WorkoutType: "cardio"},
		{UserId: "u1", LoggedAt: today, DurationMin: 30, WorkoutType: "invalid"},
		{UserId: "u1", LoggedAt: today, DurationMin: 30, WorkoutType: "cardio", Intensity: ptrString("extreme")},
		{UserId: "u1", LoggedAt: today, DurationMin: 30, WorkoutType: "cardio", CaloriesBurned: ptrInt(-1)},
		{UserId: "u1", LoggedAt: today, DurationMin: 30, WorkoutType: "cardio", Note: ptrString(strings.Repeat("x", 501))},
	}

	for i, tc := range cases {
		_, err := svc.CreateWorkout(context.Background(), tc, today)
		if !errors.Is(err, ErrWorkoutInputInvalid) {
			t.Fatalf("case %d: expected ErrWorkoutInputInvalid, got %v", i, err)
		}
	}
}

func TestCreateWorkout_NormalizesInput(t *testing.T) {
	var captured WorkoutCreate
	svc := &service{
		repo: stubRepo{
			createWorkoutFn: func(ctx context.Context, w WorkoutCreate) (*Workout, error) {
				captured = w
				return &Workout{Id: 1, LoggedAt: w.LoggedAt.Format("2006-01-02")}, nil
			},
		},
	}
	today := mustDate(t, "2026-02-21")

	note := "   after run   "
	intensity := " Medium "
	_, err := svc.CreateWorkout(context.Background(), WorkoutCreate{
		UserId:      "u1",
		LoggedAt:    today,
		DurationMin: 50,
		WorkoutType: " Running ",
		Intensity:   &intensity,
		Note:        &note,
	}, today)
	if err != nil {
		t.Fatalf("CreateWorkout returned error: %v", err)
	}

	if captured.WorkoutType != "running" {
		t.Fatalf("expected workoutType normalized to running, got %s", captured.WorkoutType)
	}
	if captured.Intensity == nil || *captured.Intensity != "medium" {
		t.Fatalf("expected intensity normalized to medium, got %v", captured.Intensity)
	}
	if captured.Note == nil || *captured.Note != "after run" {
		t.Fatalf("expected note trimmed, got %v", captured.Note)
	}
}

func ptrDate(value time.Time) *time.Time {
	return &value
}

func ptrString(value string) *string {
	return &value
}

func ptrInt(value int) *int {
	return &value
}

func findTimelineDay(t *testing.T, timeline *CycleTimeline, date string) CycleTimelineDay {
	t.Helper()
	for _, day := range timeline.Days {
		if day.Date == date {
			return day
		}
	}
	t.Fatalf("timeline day %s not found", date)
	return CycleTimelineDay{}
}
