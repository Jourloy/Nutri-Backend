package plan

import "time"

type Plan struct {
	Id                int64     `json:"id" db:"id"`
	Code              string    `json:"code" db:"code"`
	Name              string    `json:"name" db:"name"`
	PlanType          string    `json:"plan_type" db:"plan_type"`
	Version           int       `json:"version" db:"version"`
	Currency          string    `json:"currency" db:"currency"`
	AmountMinor       int64     `json:"amount_minor" db:"amount_minor"`
	BillingPeriod     string    `json:"billing_period" db:"billing_period"`
	TrialDays         int       `json:"trial_days" db:"trial_days"`
	ClientLimit       int       `json:"client_limit" db:"client_limit"`
	IsActive          bool      `json:"is_active" db:"is_active"`
	CreatedAt         time.Time `json:"-" db:"created_at"`
	UpdatedAt         time.Time `json:"-" db:"updated_at"`
	ExternalProductId *string   `json:"external_product_id,omitempty" db:"external_product_id"`
	ExternalPriceId   *string   `json:"external_price_id,omitempty" db:"external_price_id"`
}

type PlanCreate struct {
	Code            string  `json:"code" db:"code"`
	Name            string  `json:"name" db:"name"`
	PlanType        string  `json:"plan_type" db:"plan_type"`
	Version         int     `json:"version" db:"version"`
	Currency        string  `json:"currency" db:"currency"`
	AmountMinor     int64   `json:"amount_minor" db:"amount_minor"`
	BillingPeriod   string  `json:"billing_period" db:"billing_period"`
	TrialDays       int     `json:"trial_days" db:"trial_days"`
	ClientLimit     int     `json:"client_limit" db:"client_limit"`
	IsActive        bool    `json:"is_active" db:"is_active"`
	ExternalProduct *string `json:"external_product_id,omitempty" db:"external_product_id"`
	ExternalPrice   *string `json:"external_price_id,omitempty" db:"external_price_id"`
}
