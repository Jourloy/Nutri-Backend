package order

import (
    "context"
    "time"

    "github.com/charmbracelet/log"
    "github.com/jourloy/nutri-backend/internal/subscription"
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
    return nil
}

