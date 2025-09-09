package order

import (
    "context"
    "database/sql"
    "errors"
    "fmt"
    "strconv"
    "time"

    "github.com/jourloy/nutri-backend/internal/lib"
    "github.com/jourloy/nutri-backend/internal/plan"
    "github.com/jourloy/nutri-backend/internal/subscription"
    userpkg "github.com/jourloy/nutri-backend/internal/user"
    "strings"
)

type Service interface {
    Init(ctx context.Context, userId string, planId int64, email string, returnURL *string) (*InitResponse, error)
    HandleTBankWebhook(ctx context.Context, w TBankWebhook) error
    FinalizeReturn(ctx context.Context, localOrderId int64) (bool, error)
    List(ctx context.Context, userId string, isAdmin bool) ([]Order, error)
    Delete(ctx context.Context, id int64, userId string, isAdmin bool) error
    EnsureStart(ctx context.Context, userId string) (*subscription.Subscription, bool, error)
}

type service struct {
	repo     Repository
	planRepo plan.Repository
	subRepo  subscription.Repository
	tbank    TBankClient
	userRepo userpkg.Repository
}

func NewService() Service {
	return &service{
		repo:     NewRepository(),
		planRepo: plan.NewRepository(),
		subRepo:  subscription.NewRepository(),
		tbank:    NewTBankClient(),
		userRepo: userpkg.NewRepository(),
	}
}

func (s *service) Init(ctx context.Context, userId string, planId int64, email string, returnURL *string) (*InitResponse, error) {
	// Best-effort: persist email to the user's profile if provided
	if email != "" {
		_, _ = s.userRepo.UpdateEmail(ctx, userId, email)
	}
	plans, err := s.planRepo.GetAllActive(ctx)
	if err != nil {
		return nil, err
	}

	var pl *plan.Plan
	for i := range plans {
		if plans[i].Id == planId {
			pl = &plans[i]
			break
		}
	}
	if pl == nil {
		return nil, errors.New("plan not found")
	}

	placeholder := Order{Status: "pending", UserId: userId, PlanId: planId, AmountMinor: pl.AmountMinor, Currency: pl.Currency}
	created, err := s.repo.Create(ctx, placeholder)
	if err != nil {
		return nil, err
	}

    localOrderId := strconv.FormatInt(created.Id, 10)
    // Backend generates SuccessURL to return to our endpoint
    my := lib.Config.MyURL
    if my != "" && !strings.HasPrefix(my, "http://") && !strings.HasPrefix(my, "https://") {
        my = "http://" + my
    }
    successURL := fmt.Sprintf("%s/order/paid?oid=%s", my, localOrderId)
    paymentURL, tbOrderId, err := s.tbank.Init(pl.AmountMinor, localOrderId, userId, fmt.Sprintf("План %s", pl.Code), email, &successURL, true)
	if err != nil {
		msg := err.Error()
		created.LastError = &msg
		_, _ = s.repo.Update(ctx, *created)
		return nil, err
	}

	created.TbOrderId = &tbOrderId
	created.PaymentURL = &paymentURL
	if _, err := s.repo.Update(ctx, *created); err != nil {
		return nil, err
	}

    return &InitResponse{PaymentURL: paymentURL, OrderId: tbOrderId}, nil
}

// FinalizeReturn verifies payment with TBank and grants subscription, returns whether success
func (s *service) FinalizeReturn(ctx context.Context, localOrderId int64) (bool, error) {
    o, err := s.repo.GetById(ctx, localOrderId)
    if err != nil || o == nil {
        return false, err
    }
    if o.TbOrderId == nil || *o.TbOrderId == "" {
        return false, errors.New("tb order id missing")
    }
    // Assume success if user returned via SuccessURL
    ok := true
    var rebillId *string = nil
    wh := TBankWebhook{OrderId: *o.TbOrderId, Success: ok, RebillId: rebillId}
    if err := s.HandleTBankWebhook(ctx, wh); err != nil {
        return false, err
    }
    return ok, nil
}

func addMonths(t time.Time, months int) time.Time {
	// safe month add: keep day, clamp to end of month
	year := t.Year()
	month := int(t.Month()) + months
	year += (month - 1) / 12
	month = (month-1)%12 + 1
	day := t.Day()
	lastDay := time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, t.Location()).Day()
	if day > lastDay {
		day = lastDay
	}
	return time.Date(year, time.Month(month), day, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
}

func monthsForBillingPeriod(p string) int {
	switch p {
	case "year":
		return 12
	default:
		return 1
	}
}

func (s *service) HandleTBankWebhook(ctx context.Context, w TBankWebhook) error {
	if w.OrderId == "" {
		return errors.New("no orderId")
	}
	o, err := s.repo.GetByTbOrderId(ctx, w.OrderId)
	if err != nil {
		return err
	}
	if o == nil {
		return errors.New("order not found")
	}
	now := time.Now()
	if !w.Success {
		o.Status = "failed"
		o.PaidAt = nil
		_, _ = s.repo.Update(ctx, *o)
		return nil
	}
	o.Status = "paid"
	o.PaidAt = &now
	if w.RebillId != nil {
		o.TbRebillId = w.RebillId
	}
	if _, err := s.repo.Update(ctx, *o); err != nil {
		return err
	}

	// grant subscription
	plans, err := s.planRepo.GetAllActive(ctx)
	if err != nil {
		return err
	}
	var pl *plan.Plan
	for i := range plans {
		if plans[i].Id == o.PlanId {
			pl = &plans[i]
			break
		}
	}
	if pl == nil {
		return errors.New("plan not found")
	}

	// create or replace user's subscription
	periodStart := now
	periodEnd := addMonths(periodStart, monthsForBillingPeriod(pl.BillingPeriod))

	cur, _ := s.subRepo.GetByUser(ctx, o.UserId)
	if cur == nil {
		sc := subscription.SubscriptionCreate{
			PlanId:               o.PlanId,
			Status:               "active",
			PeriodStart:          periodStart,
			PeriodEnd:            periodEnd,
			AmountMinor:          pl.AmountMinor,
			Currency:             pl.Currency,
			BillingPeriod:        pl.BillingPeriod,
			ExternalSubscription: w.RebillId,
			UserId:               o.UserId,
		}
		if _, err := s.subRepo.Create(ctx, sc); err != nil {
			return err
		}
	} else {
		cur.PlanId = o.PlanId
		cur.Status = "active"
		cur.PeriodStart = periodStart
		cur.PeriodEnd = periodEnd
		cur.AmountMinor = pl.AmountMinor
		cur.Currency = pl.Currency
		cur.BillingPeriod = pl.BillingPeriod
		if w.RebillId != nil {
			cur.ExternalSubscription = w.RebillId
		}
		if _, err := s.subRepo.Update(ctx, *cur); err != nil {
			return err
		}
	}
	return nil
}

func (s *service) List(ctx context.Context, userId string, isAdmin bool) ([]Order, error) {
	return s.repo.GetAll(ctx, userId, isAdmin)
}

func (s *service) Delete(ctx context.Context, id int64, userId string, isAdmin bool) error {
	return s.repo.Delete(ctx, id, userId, isAdmin)
}

// EnsureStart checks if the user has any subscriptions; if not, grants the "start" plan.
// Returns the subscription (existing or created) and whether a new one was created.
func (s *service) EnsureStart(ctx context.Context, userId string) (*subscription.Subscription, bool, error) {
	// Check if user already has a subscription
	cur, err := s.subRepo.GetByUser(ctx, userId)
	if err == nil && cur != nil {
		return cur, false, nil
	}
	if err != nil && err != sql.ErrNoRows {
		return nil, false, err
	}

	// Find start plan
	plans, err := s.planRepo.GetAllActive(ctx)
	if err != nil {
		return nil, false, err
	}
	var pl *plan.Plan
	for i := range plans {
		if plans[i].Code == "START" {
			pl = &plans[i]
			break
		}
	}
	if pl == nil {
		return nil, false, errors.New("start plan not found")
	}

	now := time.Now()
	periodStart := now
	periodEnd := addMonths(periodStart, monthsForBillingPeriod(pl.BillingPeriod))
	sc := subscription.SubscriptionCreate{
		PlanId:        pl.Id,
		Status:        "active",
		PeriodStart:   periodStart,
		PeriodEnd:     periodEnd,
		AmountMinor:   pl.AmountMinor,
		Currency:      pl.Currency,
		BillingPeriod: pl.BillingPeriod,
		UserId:        userId,
	}
	created, err := s.subRepo.Create(ctx, sc)
	if err != nil {
		return nil, false, err
	}
	return created, true, nil
}
