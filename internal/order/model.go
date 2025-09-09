package order

import "time"

type Order struct {
	Id          int64      `json:"id" db:"id"`
	Status      string     `json:"status" db:"status"`
	UserId      string     `json:"userId" db:"user_id"`
	PlanId      int64      `json:"planId" db:"plan_id"`
	AmountMinor int64      `json:"amountMinor" db:"amount_minor"`
	Currency    string     `json:"currency" db:"currency"`
	TbOrderId   *string    `json:"tbOrderId,omitempty" db:"tb_order_id"`
	TbRebillId  *string    `json:"tbRebillId,omitempty" db:"tb_rebill_id"`
	PaymentURL  *string    `json:"paymentUrl,omitempty" db:"payment_url"`
	PaidAt      *time.Time `json:"paidAt,omitempty" db:"paid_at"`
	LastError   *string    `json:"lastError,omitempty" db:"last_error"`
	CreatedAt   time.Time  `json:"-" db:"created_at"`
	UpdatedAt   time.Time  `json:"-" db:"updated_at"`
}

type InitPayload struct {
	PlanId    int64   `json:"planId"`
	Email     string  `json:"email"`
	ReturnURL *string `json:"returnUrl,omitempty"`
}

type InitResponse struct {
	PaymentURL string `json:"paymentUrl"`
	OrderId    string `json:"orderId"`
}

// Webhook payload from TBank (simplified)
type TBankWebhook struct {
	Status   string  `json:"Status"`
	OrderId  string  `json:"OrderId"`
	RebillId *string `json:"RebillId,omitempty"`
	Success  bool    `json:"Success"`
}

type Receipt struct {
	Items    []Item `json:"Items"`
	Email    string `json:"Email"`
	Taxation string `json:"Taxation"`
}

type Item struct {
	Name     string `json:"Name"`
	Price    int64  `json:"Price"`
	Quantity int64  `json:"Quantity"`
	Amount   int64  `json:"Amount"`
	Tax      string `json:"Tax"`
}
