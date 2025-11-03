package order

import (
	"encoding/json"
	"time"
)

type Order struct {
	Id          int64      `json:"id" db:"id"`
	Status      string     `json:"status" db:"status"`
	UserId      string     `json:"userId" db:"user_id"`
	PlanId      int64      `json:"planId" db:"plan_id"`
	AmountMinor int64      `json:"amountMinor" db:"amount_minor"`
	Currency    string     `json:"currency" db:"currency"`
	TbOrderId   *string    `json:"tbOrderId,omitempty" db:"tb_order_id"`
	TbRebillId  *string    `json:"tbRebillId,omitempty" db:"tb_rebill_id"`
	CpOrderId   *string    `json:"cpOrderId,omitempty" db:"cp_order_id"`
	CpTransId   *string    `json:"cpTransactionId,omitempty" db:"cp_transaction_id"`
	CpSubId     *string    `json:"cpSubscriptionId,omitempty" db:"cp_subscription_id"`
	Provider    string     `json:"provider" db:"provider"`
	PaymentURL  *string    `json:"paymentUrl,omitempty" db:"payment_url"`
	PaidAt      *time.Time `json:"paidAt,omitempty" db:"paid_at"`
	LastError   *string    `json:"lastError,omitempty" db:"last_error"`
	AdCode      *string    `json:"adCode,omitempty" db:"ad_code"`
	CreatedAt   time.Time  `json:"-" db:"created_at"`
	UpdatedAt   time.Time  `json:"-" db:"updated_at"`
}

type CloudPaymentsNotification struct {
	Id             int64           `json:"id" db:"id"`
	Type           string          `json:"type" db:"type"`
	InvoiceId      *string         `json:"invoiceId,omitempty" db:"invoice_id"`
	OrderId        *int64          `json:"orderId,omitempty" db:"order_id"`
	SubscriptionId *string         `json:"subscriptionId,omitempty" db:"subscription_id"`
	Payload        json.RawMessage `json:"payload" db:"payload"`
	Headers        json.RawMessage `json:"headers,omitempty" db:"headers"`
	CreatedAt      time.Time       `json:"createdAt" db:"created_at"`
}

type CloudPaymentsNotificationCreate struct {
	Type           string  `db:"type"`
	InvoiceId      *string `db:"invoice_id"`
	OrderId        *int64  `db:"order_id"`
	SubscriptionId *string `db:"subscription_id"`
	Payload        string  `db:"payload"`
	Headers        string  `db:"headers"`
}

type InitPayload struct {
	PlanId    int64   `json:"planId"`
	Email     string  `json:"email"`
	ReturnURL *string `json:"returnUrl,omitempty"`
	AdCode    *string `json:"adCode,omitempty"`
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
