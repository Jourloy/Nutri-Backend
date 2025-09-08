package analytics

import "time"

type Day struct {
    Date     time.Time `json:"date"`
    Calories float64   `json:"calories"`
    Protein  float64   `json:"protein"`
    Fat      float64   `json:"fat"`
    Carbs    float64   `json:"carbs"`
}

type SeriesResponse struct {
    Days          []Day  `json:"days"`
    AllowedDays   int    `json:"allowedDays"`
    Clamped       bool   `json:"clamped"`
    PlanType      string `json:"planType"`
    RangeStart    string `json:"rangeStart"`
    RangeEnd      string `json:"rangeEnd"`
}

