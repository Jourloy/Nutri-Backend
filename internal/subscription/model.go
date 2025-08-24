package subscription

import "time"

type Subscription struct {
	Id                   int64      `json:"id" db:"id"`
	UserId               string     `json:"user_id" db:"user_id"`
	PlanId               int64      `json:"plan_id" db:"plan_id"`
	Status               string     `json:"status" db:"status"`
	PeriodStart          time.Time  `json:"period_start" db:"period_start"`
	PeriodEnd            time.Time  `json:"period_end" db:"period_end"`
	CancelAt             *time.Time `json:"cancel_at,omitempty" db:"cancel_at"`
	CanceledAt           *time.Time `json:"canceled_at,omitempty" db:"canceled_at"`
	TrialEnd             *time.Time `json:"trial_end,omitempty" db:"trial_end"`
	AmountMinor          int64      `json:"amount_minor" db:"amount_minor"`
	Currency             string     `json:"currency" db:"currency"`
	BillingPeriod        string     `json:"billing_period" db:"billing_period"`
	ExternalSubscription *string    `json:"external_subscription_id,omitempty" db:"external_subscription_id"`
	ExternalCustomer     *string    `json:"external_customer_id,omitempty" db:"external_customer_id"`
	CreatedAt            time.Time  `json:"-" db:"created_at"`
	UpdatedAt            time.Time  `json:"-" db:"updated_at"`
}

type SubscriptionCreate struct {
	PlanId               int64      `json:"plan_id" db:"plan_id"`
	Status               string     `json:"status" db:"status"`
	PeriodStart          time.Time  `json:"period_start" db:"period_start"`
	PeriodEnd            time.Time  `json:"period_end" db:"period_end"`
	CancelAt             *time.Time `json:"cancel_at,omitempty" db:"cancel_at"`
	CanceledAt           *time.Time `json:"canceled_at,omitempty" db:"canceled_at"`
	TrialEnd             *time.Time `json:"trial_end,omitempty" db:"trial_end"`
	AmountMinor          int64      `json:"amount_minor" db:"amount_minor"`
	Currency             string     `json:"currency" db:"currency"`
	BillingPeriod        string     `json:"billing_period" db:"billing_period"`
	ExternalSubscription *string    `json:"external_subscription_id,omitempty" db:"external_subscription_id"`
	ExternalCustomer     *string    `json:"external_customer_id,omitempty" db:"external_customer_id"`
	UserId               string     `json:"-" db:"user_id"`
}
