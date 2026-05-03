package promo

import "time"

// PromoCode представляет промокод
type PromoCode struct {
	Id              int64      `json:"id" db:"id"`
	Code            string     `json:"code" db:"code"`
	Description     *string    `json:"description,omitempty" db:"description"`
	DiscountType    string     `json:"discount_type" db:"discount_type"` // 'percent' или 'fixed'
	DiscountValue   int64      `json:"discount_value" db:"discount_value"`
	MaxUses         int        `json:"max_uses" db:"max_uses"`
	CurrentUses     int        `json:"current_uses" db:"current_uses"`
	ValidFrom       time.Time  `json:"valid_from" db:"valid_from"`
	ValidUntil      *time.Time `json:"valid_until,omitempty" db:"valid_until"`
	IsActive        bool       `json:"is_active" db:"is_active"`
	ApplicablePlans []int64    `json:"applicable_plans,omitempty" db:"applicable_plans"`
	MinAmountMinor  int64      `json:"min_amount_minor" db:"min_amount_minor"`
	CreatedBy       *string    `json:"created_by,omitempty" db:"created_by"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
}

// PromoCodeCreate представляет структуру для создания промокода
type PromoCodeCreate struct {
	Code            string     `json:"code" db:"code"`
	Description     *string    `json:"description,omitempty" db:"description"`
	DiscountType    string     `json:"discount_type" db:"discount_type"`
	DiscountValue   int64      `json:"discount_value" db:"discount_value"`
	MaxUses         int        `json:"max_uses" db:"max_uses"`
	ValidFrom       *time.Time `json:"valid_from,omitempty" db:"valid_from"`
	ValidUntil      *time.Time `json:"valid_until,omitempty" db:"valid_until"`
	ApplicablePlans []int64    `json:"applicable_plans,omitempty" db:"applicable_plans"`
	MinAmountMinor  int64      `json:"min_amount_minor" db:"min_amount_minor"`
}

// ValidatePromoRequest представляет запрос на валидацию промокода
type ValidatePromoRequest struct {
	Code        string `json:"code"`
	PlanId      int64  `json:"plan_id"`
	AmountMinor int64  `json:"amount_minor"`
}

// ValidatePromoResponse представляет ответ с расчетом скидки
type ValidatePromoResponse struct {
	Valid               bool   `json:"valid"`
	PromoCodeId         int64  `json:"promo_code_id,omitempty"`
	DiscountAmountMinor int64  `json:"discount_amount_minor"`
	FinalAmountMinor    int64  `json:"final_amount_minor"`
	Message             string `json:"message,omitempty"`
}
