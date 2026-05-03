package order

import (
	"context"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/jourloy/nutri02/internal/plan"
	"github.com/jourloy/nutri02/internal/subscription"
)

func StartWorker() {
	go func() {
		logger := log.WithPrefix("[ordw]")
		subRepo := subscription.NewRepository()
		tbank := NewTBankClient()
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			// run immediately on start and then every 24h
			_ = runRenewals(context.Background(), logger, subRepo, tbank)
			<-ticker.C
		}
	}()
}

func runRenewals(ctx context.Context, logger *log.Logger, subRepo subscription.Repository, tbank TBankClient) error {
	subs, err := subRepo.GetAll(ctx)
	if err != nil {
		return err
	}
	now := time.Now()
	for _, s := range subs {
		if s.Status != "active" {
			continue
		}
		if s.PeriodEnd.After(now) {
			continue
		}
		if s.ExternalSubscription == nil || *s.ExternalSubscription == "" {
			continue
		}
		if strings.HasPrefix(strings.ToLower(*s.ExternalSubscription), "sc_") {
			// CloudPayments subscriptions обрабатываются их системой
			continue
		}
		orderId := time.Now().Format("20060102") + "-renew-" + s.UserId
		if err := tbank.Charge(*s.ExternalSubscription, s.AmountMinor, orderId); err != nil {
			logger.Warn("charge failed", "user", s.UserId, "err", err)
			// mark past_due
			s.Status = "past_due"
			if _, uerr := subRepo.Update(ctx, s); uerr != nil {
				logger.Error("update sub", "err", uerr)
			}
			continue
		}
		// extend period
		months := 1
		if s.BillingPeriod == "year" {
			months = 12
		}
		s.PeriodStart = now
		s.PeriodEnd = addMonths(now, months)
		if _, uerr := subRepo.Update(ctx, s); uerr != nil {
			logger.Error("update sub", "err", uerr)
		}
	}

	// Страховка: понизить подписки с истекшим периодом без external_subscription_id
	planRepo := plan.NewRepository()
	startPlan, err := planRepo.GetByCode(ctx, "START")
	if err != nil {
		logger.Warn("failed to get START plan for downgrade check", "err", err)
		return nil
	}

	for _, s := range subs {
		// Пропустить подписки с действующим периодом
		if s.PeriodEnd.After(now) {
			continue
		}
		// Пропустить уже на START плане
		if s.PlanId == startPlan.Id {
			continue
		}
		// Пропустить подписки с активной связью с CloudPayments (они обрабатываются webhook-ами)
		if s.ExternalSubscription != nil && *s.ExternalSubscription != "" {
			continue
		}
		// Понизить до START
		logger.Info("downgrading expired subscription", "user", s.UserId, "old_plan_id", s.PlanId)
		s.PlanId = startPlan.Id
		s.AmountMinor = startPlan.AmountMinor
		s.Currency = startPlan.Currency
		s.BillingPeriod = startPlan.BillingPeriod
		s.Status = "active"
		s.PeriodStart = now
		s.PeriodEnd = addMonths(now, 1)
		if _, uerr := subRepo.Update(ctx, s); uerr != nil {
			logger.Error("downgrade sub", "err", uerr)
		}
	}

	return nil
}
