package orders

import "time"

type Order struct {
	Id          int64     `json:"id" db:"id"`
	UserId      string    `json:"user_id" db:"user_id"`
	PlanId      int64     `json:"plan_id" db:"plan_id"`
	AmountMinor int64     `json:"amount_minor" db:"amount_minor"`
	Currency    string    `json:"currency" db:"currency"`
	Status      string    `json:"status" db:"status"`
	PaymentId   *string   `json:"payment_id,omitempty" db:"payment_id"`
	PaymentURL  *string   `json:"payment_url,omitempty" db:"payment_url"`
	CreatedAt   time.Time `json:"-" db:"created_at"`
	UpdatedAt   time.Time `json:"-" db:"updated_at"`
}

type OrderCreateRequest struct {
	Name          string `json:"name"`
	BillingPeriod string `json:"billing_period"`
}

type OrderCreate struct {
	UserId      string `db:"user_id"`
	PlanId      int64  `db:"plan_id"`
	AmountMinor int64  `db:"amount_minor"`
	Currency    string `db:"currency"`
	Status      string `db:"status"`
}

type OrderResponse struct {
	PaymentURL string `json:"payment_url"`
}
