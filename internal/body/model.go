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
    Id        int64      `json:"id" db:"id"`
    UserId    string     `json:"-" db:"user_id"`
    Chest     *float64   `json:"chest,omitempty" db:"chest"`
    Waist     *float64   `json:"waist,omitempty" db:"waist"`
    Hips      *float64   `json:"hips,omitempty" db:"hips"`
    LoggedAt  time.Time  `json:"loggedAt" db:"logged_at"`
    CreatedAt time.Time  `json:"-" db:"created_at"`
    UpdatedAt time.Time  `json:"-" db:"updated_at"`
}

type MeasurementCreate struct {
    UserId   string     `db:"user_id"`
    Chest    *float64   `db:"chest"`
    Waist    *float64   `db:"waist"`
    Hips     *float64   `db:"hips"`
    LoggedAt time.Time  `db:"logged_at"`
}

type PlateauResult struct {
    IsPlateau        bool     `json:"isPlateau"`
    Goal             string   `json:"goal"`
    WindowStart      string   `json:"windowStart"`
    WindowEnd        string   `json:"windowEnd"`
    WindowDays       int      `json:"windowDays"`
    DaysWithWeight   int      `json:"daysWithWeight"`
    SlopeWeeklyPct   float64  `json:"slopeWeeklyPct"`
    DeltaKg          float64  `json:"deltaKg"`
    CaloriesGoodDays int      `json:"caloriesGoodDays"`
    ProteinGoodDays  int      `json:"proteinGoodDays"`
    CaloriesTarget   float64  `json:"caloriesTarget"`
    ProteinPerKg     float64  `json:"proteinPerKgTarget"`
    StepsAvg         float64  `json:"stepsAvg"`
    StepsTarget      int      `json:"stepsTarget"`
    SleepAvgHours    float64  `json:"sleepAvgHours"`
    Reason           string   `json:"reason"`
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
