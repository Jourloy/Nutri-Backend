package fit

import "time"

type FitProfile struct {
	Id            string     `json:"id"`
	Age           int64      `json:"age"`
	Gender        string     `json:"gender"`
	Height        int64      `json:"height"`
	Weight        int64      `json:"weight"`
	ActivityLevel float64    `json:"activityLevel"`
	Goal          string     `json:"goal"`
	Calories      float64    `json:"calories"`
	Protein       float64    `json:"protein"`
	Fat           float64    `json:"fat"`
	Carbs         float64    `json:"carbs"`
	UserId        string     `json:"-"`
	CreatedAt     time.Time  `json:"-"`
	UpdatedAt     time.Time  `json:"-"`
	DeletedAt     *time.Time `json:"-"`
}

type FitProfileCreate struct {
	Age           int64   `json:"age"`
	Gender        string  `json:"gender"`
	Height        int64   `json:"height"`
	Weight        int64   `json:"weight"`
	ActivityLevel float64 `json:"activityLevel"`
	Goal          string  `json:"goal"`
	Calories      float64 `json:"calories"`
	Protein       float64 `json:"protein"`
	Fat           float64 `json:"fat"`
	Carbs         float64 `json:"carbs"`
	UserId        string
}
