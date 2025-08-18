package product

import "time"

type Product struct {
	Id        int64     `json:"id"`
	Name      string    `json:"name"`
	Amount    int64     `json:"amount"`
	Unit      string    `json:"unit"`
	Calories  int64     `json:"calories"`
	Protein   int64     `json:"protein"`
	Fat       int64     `json:"fat"`
	Carbs     int64     `json:"carbs"`
	UserId    string    `json:"-"`
	FitId     string    `json:"-"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

type ProductCreate struct {
	Name     string `json:"name"`
	Amount   int64  `json:"amount"`
	Unit     string `json:"unit"`
	Calories int64  `json:"calories"`
	Protein  int64  `json:"protein"`
	Fat      int64  `json:"fat"`
	Carbs    int64  `json:"carbs"`
	UserId   string `json:"-"`
	FitId    string `json:"-"`
}
