package fit

import "time"

type FitProfile struct {
	Id            string     `json:"id" db:"id"`
	Age           int64      `json:"age" db:"age"`
	Gender        string     `json:"gender" db:"gender"`
	Height        int64      `json:"height" db:"height"`
	Weight        int64      `json:"weight" db:"weight"`
	ActivityLevel float64    `json:"activityLevel" db:"activity_level"`
	Goal          string     `json:"goal" db:"goal"`
	Calories      float64    `json:"calories" db:"calories"`
	Protein       float64    `json:"protein" db:"protein"`
	Fat           float64    `json:"fat" db:"fat"`
	Carbs         float64    `json:"carbs" db:"carbs"`
	WaterLimit    *int64     `json:"waterLimit" db:"water_limit"`
	UserId        string     `json:"-" db:"user_id"`
	CreatedAt     time.Time  `json:"-" db:"created_at"`
	UpdatedAt     time.Time  `json:"-" db:"updated_at"`
	DeletedAt     *time.Time `json:"-" db:"deleted_at"`
}

type FitProfileCreate struct {
	Age           int64   `json:"age" db:"age"`
	Gender        string  `json:"gender" db:"gender"`
	Height        int64   `json:"height" db:"height"`
	Weight        int64   `json:"weight" db:"weight"`
	ActivityLevel float64 `json:"activityLevel" db:"activity_level"`
	Goal          string  `json:"goal" db:"goal"`
	Calories      float64 `json:"calories" db:"calories"`
	Protein       float64 `json:"protein" db:"protein"`
	Fat           float64 `json:"fat" db:"fat"`
	Carbs         float64 `json:"carbs" db:"carbs"`
	WaterLimit    int64   `json:"waterLimit" db:"water_limit"`
	UserId        string  `json:"-" db:"user_id"`
}
