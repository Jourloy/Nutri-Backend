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
	Calories      int64      `json:"calories"`
	Protein       int64      `json:"protein"`
	Fat           int64      `json:"fat"`
	Carbs         int64      `json:"carbs"`
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
	Calories      int64   `json:"calories"`
	Protein       int64   `json:"protein"`
	Fat           int64   `json:"fat"`
	Carbs         int64   `json:"carbs"`
	UserId        string
}
