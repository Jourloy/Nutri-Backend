package product

import "time"

type Product struct {
	Id        int64     `json:"id"`
	Name      string    `json:"name"`
	Amount    int64     `json:"amount"`
	Unit      string    `json:"unit"`
	Calories  float64   `json:"calories"`
	Protein   float64   `json:"protein"`
	Fat       float64   `json:"fat"`
	Carbs     float64   `json:"carbs"`
	UserId    string    `json:"-"`
	FitId     string    `json:"-"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

type ProductCreate struct {
	Name     string  `json:"name"`
	Amount   int64   `json:"amount"`
	Unit     string  `json:"unit"`
	Calories float64 `json:"calories"`
	Protein  float64 `json:"protein"`
	Fat      float64 `json:"fat"`
	Carbs    float64 `json:"carbs"`
	UserId   string  `json:"-"`
	FitId    string  `json:"-"`
}
