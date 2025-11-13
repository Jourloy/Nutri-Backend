package promo

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/jourloy/nutri-backend/internal/database"
)

type Repository interface {
	CreatePromoCode(ctx context.Context, createdBy string, promo *PromoCodeCreate) (*PromoCode, error)
	GetPromoCode(ctx context.Context, code string) (*PromoCode, error)
	GetPromoCodeById(ctx context.Context, id int64) (*PromoCode, error)
	GetAllPromoCodes(ctx context.Context, limit, offset int) ([]PromoCode, error)
	UpdatePromoCode(ctx context.Context, id int64, promo *PromoCodeCreate) (*PromoCode, error)
	DeletePromoCode(ctx context.Context, id int64) error
	IncrementUsage(ctx context.Context, id int64) error
	ValidateAndCalculate(ctx context.Context, code string, planId, amountMinor int64) (*ValidatePromoResponse, error)
}

type repository struct {
	db *sqlx.DB
}

func NewRepository() Repository {
	return &repository{db: database.Database}
}

const promoColumns = `
	id, code, description, discount_type, discount_value,
	max_uses, current_uses, valid_from, valid_until,
	is_active, applicable_plans, min_amount_minor,
	created_by, created_at, updated_at
`

func (r *repository) CreatePromoCode(ctx context.Context, createdBy string, promo *PromoCodeCreate) (*PromoCode, error) {
	validFrom := time.Now()
	if promo.ValidFrom != nil {
		validFrom = *promo.ValidFrom
	}

	const q = `
		INSERT INTO promo_codes (
			code, description, discount_type, discount_value,
			max_uses, valid_from, valid_until, applicable_plans,
			min_amount_minor, created_by
		) VALUES (
			:code, :description, :discount_type, :discount_value,
			:max_uses, :valid_from, :valid_until, :applicable_plans,
			:min_amount_minor, :created_by
		)
		RETURNING ` + promoColumns

	args := map[string]interface{}{
		"code":             promo.Code,
		"description":      promo.Description,
		"discount_type":    promo.DiscountType,
		"discount_value":   promo.DiscountValue,
		"max_uses":         promo.MaxUses,
		"valid_from":       validFrom,
		"valid_until":      promo.ValidUntil,
		"applicable_plans": pq.Array(promo.ApplicablePlans),
		"min_amount_minor": promo.MinAmountMinor,
		"created_by":       createdBy,
	}

	rows, err := r.db.NamedQueryContext(ctx, q, args)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var pc PromoCode
		if err := rows.StructScan(&pc); err != nil {
			return nil, err
		}
		return &pc, nil
	}

	return nil, sql.ErrNoRows
}

func (r *repository) GetPromoCode(ctx context.Context, code string) (*PromoCode, error) {
	const q = `SELECT ` + promoColumns + ` FROM promo_codes WHERE code = $1`

	var pc PromoCode
	if err := r.db.GetContext(ctx, &pc, q, code); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &pc, nil
}

func (r *repository) GetPromoCodeById(ctx context.Context, id int64) (*PromoCode, error) {
	const q = `SELECT ` + promoColumns + ` FROM promo_codes WHERE id = $1`

	var pc PromoCode
	if err := r.db.GetContext(ctx, &pc, q, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &pc, nil
}

func (r *repository) GetAllPromoCodes(ctx context.Context, limit, offset int) ([]PromoCode, error) {
	const q = `
		SELECT ` + promoColumns + `
		FROM promo_codes
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	var promoCodes []PromoCode
	if err := r.db.SelectContext(ctx, &promoCodes, q, limit, offset); err != nil {
		return nil, err
	}
	return promoCodes, nil
}

func (r *repository) UpdatePromoCode(ctx context.Context, id int64, promo *PromoCodeCreate) (*PromoCode, error) {
	const q = `
		UPDATE promo_codes
		SET
			description = $2,
			discount_type = $3,
			discount_value = $4,
			max_uses = $5,
			valid_until = $6,
			applicable_plans = $7,
			min_amount_minor = $8,
			updated_at = NOW()
		WHERE id = $1
		RETURNING ` + promoColumns

	var pc PromoCode
	if err := r.db.GetContext(ctx, &pc, q,
		id,
		promo.Description,
		promo.DiscountType,
		promo.DiscountValue,
		promo.MaxUses,
		promo.ValidUntil,
		pq.Array(promo.ApplicablePlans),
		promo.MinAmountMinor,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &pc, nil
}

func (r *repository) DeletePromoCode(ctx context.Context, id int64) error {
	const q = `DELETE FROM promo_codes WHERE id = $1`
	_, err := r.db.ExecContext(ctx, q, id)
	return err
}

func (r *repository) IncrementUsage(ctx context.Context, id int64) error {
	const q = `
		UPDATE promo_codes
		SET current_uses = current_uses + 1, updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, q, id)
	return err
}

func (r *repository) ValidateAndCalculate(ctx context.Context, code string, planId, amountMinor int64) (*ValidatePromoResponse, error) {
	promo, err := r.GetPromoCode(ctx, code)
	if err != nil {
		return nil, err
	}

	if promo == nil {
		return &ValidatePromoResponse{
			Valid:   false,
			Message: "Промокод не найден",
		}, nil
	}

	// Проверка активности
	if !promo.IsActive {
		return &ValidatePromoResponse{
			Valid:   false,
			Message: "Промокод неактивен",
		}, nil
	}

	// Проверка срока действия
	now := time.Now()
	if now.Before(promo.ValidFrom) {
		return &ValidatePromoResponse{
			Valid:   false,
			Message: "Промокод еще не действителен",
		}, nil
	}

	if promo.ValidUntil != nil && now.After(*promo.ValidUntil) {
		return &ValidatePromoResponse{
			Valid:   false,
			Message: "Промокод истек",
		}, nil
	}

	// Проверка лимита использований
	if promo.MaxUses > 0 && promo.CurrentUses >= promo.MaxUses {
		return &ValidatePromoResponse{
			Valid:   false,
			Message: "Промокод исчерпан",
		}, nil
	}

	// Проверка минимальной суммы
	if amountMinor < promo.MinAmountMinor {
		return &ValidatePromoResponse{
			Valid:   false,
			Message: "Сумма заказа меньше минимальной для промокода",
		}, nil
	}

	// Проверка применимости к тарифу
	if len(promo.ApplicablePlans) > 0 {
		found := false
		for _, pid := range promo.ApplicablePlans {
			if pid == planId {
				found = true
				break
			}
		}
		if !found {
			return &ValidatePromoResponse{
				Valid:   false,
				Message: "Промокод не применим к этому тарифу",
			}, nil
		}
	}

	// Расчет скидки
	var discountAmount int64
	if promo.DiscountType == "percent" {
		discountAmount = (amountMinor * promo.DiscountValue) / 100
	} else { // fixed
		discountAmount = promo.DiscountValue
	}

	// Скидка не может быть больше суммы заказа
	if discountAmount > amountMinor {
		discountAmount = amountMinor
	}

	finalAmount := amountMinor - discountAmount

	return &ValidatePromoResponse{
		Valid:              true,
		PromoCodeId:        promo.Id,
		DiscountAmountMinor: discountAmount,
		FinalAmountMinor:   finalAmount,
		Message:            "Промокод применен успешно",
	}, nil
}
