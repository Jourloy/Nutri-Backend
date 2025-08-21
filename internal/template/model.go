package template

import "time"

type Template struct {
	Id        int64     `json:"id"`
	Name      string    `json:"name"`
	Calories  float64   `json:"calories"`
	Protein   float64   `json:"protein"`
	Fat       float64   `json:"fat"`
	Carbs     float64   `json:"carbs"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}
