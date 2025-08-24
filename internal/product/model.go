package product

import "time"

type Product struct {
	Id            int64     `json:"id" db:"id"`
	Name          string    `json:"name" db:"name"`
	Amount        int64     `json:"amount" db:"amount"`
	Unit          string    `json:"unit" db:"unit"`
	Calories      float64   `json:"calories" db:"calories"`
	Protein       float64   `json:"protein" db:"protein"`
	Fat           float64   `json:"fat" db:"fat"`
	Carbs         float64   `json:"carbs" db:"carbs"`
	BasicCalories float64   `json:"basicCalories" db:"basic_calories"`
	BasicProtein  float64   `json:"basicProtein" db:"basic_protein"`
	BasicFat      float64   `json:"basicFat" db:"basic_fat"`
	BasicCarbs    float64   `json:"basicCarbs" db:"basic_carbs"`
	IsWater       bool      `json:"isWater" db:"is_water"`
	UserId        string    `json:"-" db:"user_id"`
	FitId         string    `json:"-" db:"fit_id"`
	CreatedAt     time.Time `json:"-" db:"created_at"`
	UpdatedAt     time.Time `json:"-" db:"updated_at"`
}

type ProductCreate struct {
	Name          string  `json:"name" db:"name"`
	Amount        int64   `json:"amount" db:"amount"`
	Unit          string  `json:"unit" db:"unit"`
	Calories      float64 `json:"calories" db:"calories"`
	Protein       float64 `json:"protein" db:"protein"`
	Fat           float64 `json:"fat" db:"fat"`
	Carbs         float64 `json:"carbs" db:"carbs"`
	BasicCalories float64 `json:"basicCalories" db:"basic_calories"`
	BasicProtein  float64 `json:"basicProtein" db:"basic_protein"`
	BasicFat      float64 `json:"basicFat" db:"basic_fat"`
	BasicCarbs    float64 `json:"basicCarbs" db:"basic_carbs"`
	IsWater       bool    `json:"isWater" db:"is_water"`
	UserId        string  `json:"-" db:"user_id"`
	FitId         string  `json:"-" db:"fit_id"`
}
