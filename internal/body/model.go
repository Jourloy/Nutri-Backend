package body

import "time"

type Weight struct {
	Id        int64     `json:"id" db:"id"`
	UserId    string    `json:"-" db:"user_id"`
	Value     float64   `json:"value" db:"value"`
	LoggedAt  time.Time `json:"loggedAt" db:"logged_at"`
	CreatedAt time.Time `json:"-" db:"created_at"`
	UpdatedAt time.Time `json:"-" db:"updated_at"`
}

type WeightCreate struct {
	UserId   string    `db:"user_id"`
	Value    float64   `db:"value"`
	LoggedAt time.Time `db:"logged_at"`
}

type Measurement struct {
	Id        int64     `json:"id" db:"id"`
	UserId    string    `json:"-" db:"user_id"`
	Chest     *float64  `json:"chest,omitempty" db:"chest"`
	Waist     *float64  `json:"waist,omitempty" db:"waist"`
	Hips      *float64  `json:"hips,omitempty" db:"hips"`
	LoggedAt  time.Time `json:"loggedAt" db:"logged_at"`
	CreatedAt time.Time `json:"-" db:"created_at"`
	UpdatedAt time.Time `json:"-" db:"updated_at"`
}

type MeasurementCreate struct {
	UserId   string    `db:"user_id"`
	Chest    *float64  `db:"chest"`
	Waist    *float64  `db:"waist"`
	Hips     *float64  `db:"hips"`
	LoggedAt time.Time `db:"logged_at"`
}

type PlateauResult struct {
	IsPlateau        bool    `json:"isPlateau"`
	Goal             string  `json:"goal"`
	WindowStart      string  `json:"windowStart"`
	WindowEnd        string  `json:"windowEnd"`
	WindowDays       int     `json:"windowDays"`
	DaysWithWeight   int     `json:"daysWithWeight"`
	SlopeWeeklyPct   float64 `json:"slopeWeeklyPct"`
	DeltaKg          float64 `json:"deltaKg"`
	CaloriesGoodDays int     `json:"caloriesGoodDays"`
	ProteinGoodDays  int     `json:"proteinGoodDays"`
	CaloriesTarget   float64 `json:"caloriesTarget"`
	ProteinPerKg     float64 `json:"proteinPerKgTarget"`
	StepsAvg         float64 `json:"stepsAvg"`
	StepsTarget      int     `json:"stepsTarget"`
	SleepAvgHours    float64 `json:"sleepAvgHours"`
	Reason           string  `json:"reason"`
}

type Activity struct {
	Id        int64     `json:"id" db:"id"`
	UserId    string    `json:"-" db:"user_id"`
	Steps     *int      `json:"steps,omitempty" db:"steps"`
	SleepMin  *int      `json:"sleepMin,omitempty" db:"sleep_min"`
	LoggedAt  time.Time `json:"loggedAt" db:"logged_at"`
	CreatedAt time.Time `json:"-" db:"created_at"`
	UpdatedAt time.Time `json:"-" db:"updated_at"`
}

type ActivityCreate struct {
	UserId   string    `db:"user_id"`
	Steps    *int      `db:"steps"`
	SleepMin *int      `db:"sleep_min"`
	LoggedAt time.Time `db:"logged_at"`
}

type Workout struct {
	Id             int64      `json:"id" db:"id"`
	UserId         string     `json:"-" db:"user_id"`
	LoggedAt       string     `json:"loggedAt" db:"logged_at"`
	DurationMin    int        `json:"durationMin" db:"duration_min"`
	WorkoutType    string     `json:"workoutType" db:"workout_type"`
	Intensity      *string    `json:"intensity,omitempty" db:"intensity"`
	CaloriesBurned *int       `json:"caloriesBurned,omitempty" db:"calories_burned"`
	Note           *string    `json:"note,omitempty" db:"note"`
	CreatedAt      *time.Time `json:"-" db:"created_at"`
	UpdatedAt      *time.Time `json:"-" db:"updated_at"`
}

type WorkoutCreate struct {
	UserId         string    `db:"user_id"`
	LoggedAt       time.Time `db:"logged_at"`
	DurationMin    int       `db:"duration_min"`
	WorkoutType    string    `db:"workout_type"`
	Intensity      *string   `db:"intensity"`
	CaloriesBurned *int      `db:"calories_burned"`
	Note           *string   `db:"note"`
}

type WorkoutDailySummary struct {
	Date                  string `json:"date"`
	TotalDurationMin      int    `json:"totalDurationMin"`
	BaseWaterLimitMl      int    `json:"baseWaterLimitMl"`
	WaterBonusMl          int    `json:"waterBonusMl"`
	EffectiveWaterLimitMl int    `json:"effectiveWaterLimitMl"`
	MaxWaterLimitMl       int    `json:"maxWaterLimitMl"`
	Applied               bool   `json:"applied"`
}

type Cycle struct {
	Id        int64      `json:"id" db:"id"`
	UserId    string     `json:"-" db:"user_id"`
	StartDate time.Time  `json:"startDate" db:"start_date"`
	EndDate   *time.Time `json:"endDate,omitempty" db:"end_date"`
	CreatedAt time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time  `json:"updatedAt" db:"updated_at"`
}

type CycleDailyEvent struct {
	Id            int64   `json:"id" db:"id"`
	EventCategory string  `json:"eventCategory" db:"event_category"`
	EventCode     string  `json:"eventCode" db:"event_code"`
	Quantity      int     `json:"quantity" db:"quantity"`
	Intensity     *string `json:"intensity,omitempty" db:"intensity"`
}

type CycleDayLog struct {
	Id            int64             `json:"id"`
	LoggedAt      string            `json:"loggedAt"`
	FlowIntensity *string           `json:"flowIntensity,omitempty"`
	Note          *string           `json:"note,omitempty"`
	Events        []CycleDailyEvent `json:"events"`
}

type CycleDayUpsertInput struct {
	LoggedAt      time.Time
	FlowIntensity *string
	Note          *string
	Events        []CycleDayEventInput
}

type CycleDayEventInput struct {
	EventCategory string
	EventCode     string
	Quantity      int
	Intensity     *string
}

type CycleSeedInput struct {
	Mode     string
	LoggedAt time.Time
}

type CycleCatalogEvent struct {
	Code string `json:"code"`
}

type CycleCatalogCategory struct {
	Category string              `json:"category"`
	Events   []CycleCatalogEvent `json:"events"`
}

type CycleGoals struct {
	Calories float64 `json:"calories"`
	Protein  float64 `json:"protein"`
	Fat      float64 `json:"fat"`
	Carbs    float64 `json:"carbs"`
}

type CycleSummary struct {
	Date               string     `json:"date"`
	IsPeriodDay        bool       `json:"isPeriodDay"`
	CurrentCycleStart  *string    `json:"currentCycleStart,omitempty"`
	CurrentCycleEnd    *string    `json:"currentCycleEnd,omitempty"`
	CycleLengthDays    int        `json:"cycleLengthDays"`
	PeriodLengthDays   int        `json:"periodLengthDays"`
	PredictedPeriodEnd *string    `json:"predictedPeriodEnd,omitempty"`
	PredictedNextStart *string    `json:"predictedNextStart,omitempty"`
	Confidence         string     `json:"confidence"`
	BaseGoals          CycleGoals `json:"baseGoals"`
	EffectiveGoals     CycleGoals `json:"effectiveGoals"`
}

type CycleTimelineDay struct {
	Date            string  `json:"date"`
	Phase           string  `json:"phase"`
	PhaseSource     string  `json:"phaseSource"`
	PeriodStatus    string  `json:"periodStatus"`
	IsFertileWindow bool    `json:"isFertileWindow"`
	IsOvulationDay  bool    `json:"isOvulationDay"`
	HasLog          bool    `json:"hasLog"`
	FlowIntensity   *string `json:"flowIntensity,omitempty"`
}

type CycleTimelineSummary struct {
	CycleSummary
	Phase                  string  `json:"phase"`
	PhaseSource            string  `json:"phaseSource"`
	PeriodStatus           string  `json:"periodStatus"`
	HasCycleSeed           bool    `json:"hasCycleSeed"`
	HasActiveCycle         bool    `json:"hasActiveCycle"`
	ActiveCycleStart       *string `json:"activeCycleStart,omitempty"`
	CycleDayNumber         int     `json:"cycleDayNumber"`
	PredictedOvulationDate *string `json:"predictedOvulationDate,omitempty"`
	FertileWindowStart     *string `json:"fertileWindowStart,omitempty"`
	FertileWindowEnd       *string `json:"fertileWindowEnd,omitempty"`
	DropletFillRatio       float64 `json:"dropletFillRatio"`
	DropletState           string  `json:"dropletState"`
	DaysUntilNextPeriod    int     `json:"daysUntilNextPeriodStart"`
	DaysUntilPhaseChange   int     `json:"daysUntilPhaseChange"`
}

type CycleTimeline struct {
	Summary CycleTimelineSummary `json:"summary"`
	Days    []CycleTimelineDay   `json:"days"`
}

type PlateauEvent struct {
	Id               int64     `json:"id" db:"id"`
	UserId           string    `json:"-" db:"user_id"`
	WindowStart      time.Time `json:"windowStart" db:"window_start"`
	WindowEnd        time.Time `json:"windowEnd" db:"window_end"`
	Goal             *string   `json:"goal,omitempty" db:"goal"`
	SlopeWeeklyPct   float64   `json:"slopeWeeklyPct" db:"slope_weekly_pct"`
	DeltaKg          float64   `json:"deltaKg" db:"delta_kg"`
	DaysWithWeight   int       `json:"daysWithWeight" db:"days_with_weight"`
	CaloriesGoodDays int       `json:"caloriesGoodDays" db:"calories_good_days"`
	ProteinGoodDays  int       `json:"proteinGoodDays" db:"protein_good_days"`
	WindowDays       int       `json:"windowDays" db:"window_days"`
	IsPlateau        bool      `json:"isPlateau" db:"is_plateau"`
	Reason           string    `json:"reason" db:"reason"`
	CreatedAt        time.Time `json:"createdAt" db:"created_at"`
}

type BMIResult struct {
	CurrentWeight    float64 `json:"currentWeight"`
	Height           float64 `json:"height"`
	BMI              float64 `json:"bmi"`
	BMICategory      string  `json:"bmiCategory"`
	RecommendedMinKg float64 `json:"recommendedMinKg"`
	RecommendedMaxKg float64 `json:"recommendedMaxKg"`
	IsHealthy        bool    `json:"isHealthy"`
	DiffFromMin      float64 `json:"diffFromMin"`
	DiffFromMax      float64 `json:"diffFromMax"`
}
