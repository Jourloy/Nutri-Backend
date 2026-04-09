package order

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jourloy/somivyn/internal/lib"
	"github.com/jourloy/somivyn/internal/plan"
	"github.com/jourloy/somivyn/internal/subscription"
	userpkg "github.com/jourloy/somivyn/internal/user"
)

const (
	providerTBank         = "tbank"
	providerCloudPayments = "cloudpayments"
	taxationSystemUSN     = 1
	vatNone               = 0
)

type Service interface {
	Init(ctx context.Context, userId string, planId int64, email string, returnURL *string, adCode *string) (*InitResponse, error)
	HandleTBankWebhook(ctx context.Context, w TBankWebhook) error
	HandleCloudPaymentsNotification(ctx context.Context, notifType string, body []byte, headers map[string]string) (map[string]any, error)
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
	cp       CloudPaymentsClient
	userRepo userpkg.Repository
}

func NewService() Service {
	return &service{
		repo:     NewRepository(),
		planRepo: plan.NewRepository(),
		subRepo:  subscription.NewRepository(),
		tbank:    NewTBankClient(),
		cp:       NewCloudPaymentsClient(),
		userRepo: userpkg.NewRepository(),
	}
}

func (s *service) Init(ctx context.Context, userId string, planId int64, email string, returnURL *string, adCode *string) (*InitResponse, error) {
	// Don't save email
	// if email != "" {
	// 	_, _ = s.userRepo.UpdateEmail(ctx, userId, email)
	// }

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

	placeholder := Order{
		Status:      "pending",
		Provider:    providerCloudPayments,
		UserId:      userId,
		PlanId:      planId,
		AmountMinor: pl.AmountMinor,
		Currency:    pl.Currency,
		AdCode:      adCode,
	}
	created, err := s.repo.Create(ctx, placeholder)
	if err != nil {
		return nil, err
	}

	localOrderId := strconv.FormatInt(created.Id, 10)
	successURL := buildSuccessRedirect(localOrderId)
	failURL := buildFailRedirect()

	description := fmt.Sprintf("План %s", pl.Code)
	jsonData := buildCloudPaymentsJsonData(description, email, float64(pl.AmountMinor), pl.Currency, pl.BillingPeriod)

	orderReq := CloudPaymentsOrderRequest{
		Amount:             float64(pl.AmountMinor),
		Currency:           pl.Currency,
		Description:        description,
		Email:              email,
		AccountID:          userId,
		InvoiceID:          localOrderId,
		SuccessRedirectURL: successURL,
		FailRedirectURL:    failURL,
		SendEmail:          email != "",
		JsonData:           jsonData,
	}

	orderResp, err := s.cp.CreateOrder(ctx, orderReq)
	if err != nil {
		msg := err.Error()
		created.LastError = &msg
		_, _ = s.repo.Update(ctx, *created)
		return nil, err
	}

	created.Provider = providerCloudPayments
	created.CpOrderId = &orderResp.ID
	created.PaymentURL = &orderResp.URL
	created.LastError = nil
	if _, err := s.repo.Update(ctx, *created); err != nil {
		return nil, err
	}

	return &InitResponse{PaymentURL: orderResp.URL, OrderId: orderResp.ID}, nil
}

func (s *service) FinalizeReturn(ctx context.Context, localOrderId int64) (bool, error) {
	o, err := s.repo.GetById(ctx, localOrderId)
	if err != nil {
		return false, err
	}
	if o == nil {
		return false, errors.New("order not found")
	}
	return o.Status == "paid", nil
}

func ensureHTTP(raw string) string {
	if raw == "" {
		return raw
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return raw
	}
	return "http://" + raw
}

func buildSuccessRedirect(localOrderID string) string {
	my := ensureHTTP(lib.Config.MyURL)
	if my == "" {
		return ""
	}
	return fmt.Sprintf("%s/order/paid?oid=%s", my, localOrderID)
}

func buildFailRedirect() string {
	front := ensureHTTP(lib.Config.FrontURL)
	if front == "" {
		return ""
	}
	return front + "/prices?error=1"
}

func buildCloudPaymentsReceipt(description, email string, amount float64, currency string) map[string]any {
	items := []map[string]any{
		{
			"label":    description,
			"price":    amount,
			"quantity": 1,
			"amount":   amount,
			"vat":      vatNone,
		},
	}
	receipt := map[string]any{
		"items":          items,
		"taxationSystem": taxationSystemUSN,
		"currency":       currency,
	}
	if email != "" {
		receipt["email"] = email
	}
	return receipt
}

func buildCloudPaymentsJsonData(description, email string, amount float64, currency, billingPeriod string) map[string]any {
	receipt := buildCloudPaymentsReceipt(description, email, amount, currency)
	interval, period := cpSchedule(billingPeriod)

	return map[string]any{
		"cloudPayments": map[string]any{
			"customerReceipt": receipt,
			"recurrent": map[string]any{
				"interval": interval,
				"period":   period,
			},
		},
	}
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

func (s *service) getPlanByID(ctx context.Context, planID int64) (*plan.Plan, error) {
	plans, err := s.planRepo.GetAllActive(ctx)
	if err != nil {
		return nil, err
	}
	for i := range plans {
		if plans[i].Id == planID {
			return &plans[i], nil
		}
	}
	return nil, errors.New("plan not found")
}

func cloneStringPtr(src *string) *string {
	if src == nil || *src == "" {
		return nil
	}
	val := *src
	return &val
}

func stringPtr(value string) *string {
	if value == "" {
		return nil
	}
	v := value
	return &v
}

func parseOrderID(invoice string) *int64 {
	if invoice == "" {
		return nil
	}
	if id, err := strconv.ParseInt(invoice, 10, 64); err == nil {
		return &id
	}
	return nil
}

func cpHeadersJSON(headers map[string]string) string {
	if len(headers) == 0 {
		return "{}"
	}
	b, err := json.Marshal(headers)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func cpExtractString(data map[string]any, key string) string {
	if data == nil {
		return ""
	}
	v, ok := data[key]
	if !ok || v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case json.Number:
		return val.String()
	case float64:
		if math.Trunc(val) == val {
			return strconv.FormatInt(int64(val), 10)
		}
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		if val {
			return "true"
		}
		return "false"
	default:
		b, err := json.Marshal(val)
		if err != nil {
			return ""
		}
		return string(b)
	}
}

func cpExtractFloat(data map[string]any, key string) (float64, bool) {
	if data == nil {
		return 0, false
	}
	v, ok := data[key]
	if !ok || v == nil {
		return 0, false
	}
	switch val := v.(type) {
	case float64:
		return val, true
	case json.Number:
		f, err := val.Float64()
		if err == nil {
			return f, true
		}
	case string:
		if val == "" {
			return 0, false
		}
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f, true
		}
	}
	return 0, false
}

func cpParseDateTime(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, errors.New("empty datetime")
	}
	layout := "2006-01-02 15:04:05"
	return time.ParseInLocation(layout, value, time.UTC)
}

func cpAmountToMinor(amount float64) int64 {
	return int64(math.Round(amount))
}

func cpSchedule(period string) (string, int) {
	switch strings.ToLower(period) {
	case "week":
		return "Week", 1
	case "year":
		return "Month", 12
	default:
		return "Month", 1
	}
}

func parseCloudPaymentsPayload(body []byte, headers map[string]string) (map[string]any, error) {
	ct := ""
	if headers != nil {
		if v, ok := headers["Content-Type"]; ok {
			ct = v
		} else if v, ok := headers["content-type"]; ok {
			ct = v
		}
	}
	ct = strings.ToLower(ct)

	var data map[string]any

	switch {
	case strings.Contains(ct, "application/json"):
		if err := json.Unmarshal(body, &data); err != nil {
			return nil, err
		}
	default:
		values, err := url.ParseQuery(string(body))
		if err != nil {
			return nil, err
		}
		data = make(map[string]any, len(values))
		for key, vals := range values {
			if len(vals) == 0 {
				continue
			}
			data[key] = vals[0]
		}
	}
	return data, nil
}
func (s *service) processPaidOrder(ctx context.Context, o *Order, paidAt time.Time, externalSub *string) error {
	if o == nil {
		return errors.New("order not found")
	}
	o.Status = "paid"
	o.PaidAt = &paidAt
	if o.Provider == "" {
		if o.TbOrderId != nil && *o.TbOrderId != "" {
			o.Provider = providerTBank
		} else {
			o.Provider = providerCloudPayments
		}
	}
	if externalSub != nil && *externalSub != "" {
		switch o.Provider {
		case providerTBank:
			o.TbRebillId = cloneStringPtr(externalSub)
		default:
			o.CpSubId = cloneStringPtr(externalSub)
		}
	}
	if _, err := s.repo.Update(ctx, *o); err != nil {
		return err
	}

	pl, err := s.getPlanByID(ctx, o.PlanId)
	if err != nil {
		return err
	}

	periodStart := paidAt
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
			ExternalSubscription: cloneStringPtr(externalSub),
			UserId:               o.UserId,
			AdCode:               o.AdCode,
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
		if externalSub != nil && *externalSub != "" {
			cur.ExternalSubscription = cloneStringPtr(externalSub)
		}
		cur.AdCode = o.AdCode
		if _, err := s.subRepo.Update(ctx, *cur); err != nil {
			return err
		}
	}
	return nil
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
	if o.Provider == "" {
		o.Provider = providerTBank
	}
	var subID *string
	if w.RebillId != nil && *w.RebillId != "" {
		id := *w.RebillId
		subID = &id
	}
	return s.processPaidOrder(ctx, o, now, subID)
}

func (s *service) HandleCloudPaymentsNotification(ctx context.Context, notifType string, body []byte, headers map[string]string) (map[string]any, error) {
	nt := strings.ToLower(notifType)

	payload, err := parseCloudPaymentsPayload(body, headers)
	if err != nil {
		return nil, err
	}

	invoice := cpExtractString(payload, "InvoiceId")
	subscriptionID := cpExtractString(payload, "SubscriptionId")
	orderID := parseOrderID(invoice)

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	notif := CloudPaymentsNotificationCreate{
		Type:           nt,
		InvoiceId:      stringPtr(invoice),
		OrderId:        orderID,
		SubscriptionId: stringPtr(subscriptionID),
		Payload:        string(payloadJSON),
		Headers:        cpHeadersJSON(headers),
	}
	if _, err := s.repo.SaveNotification(ctx, notif); err != nil {
		return nil, err
	}

	switch nt {
	case "check":
		return s.handleCPCheck(ctx, payload, orderID)
	case "pay":
		return s.handleCPPay(ctx, payload, orderID, subscriptionID)
	case "fail":
		return s.handleCPFail(ctx, payload, orderID, subscriptionID)
	case "recurrent":
		return s.handleCPRecurrent(ctx, payload, subscriptionID)
	default:
		return map[string]any{"code": 0}, nil
	}
}

func (s *service) handleCPCheck(ctx context.Context, payload map[string]any, orderID *int64) (map[string]any, error) {
	if orderID == nil {
		return map[string]any{"code": 0}, nil
	}
	o, err := s.repo.GetById(ctx, *orderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return map[string]any{"code": 13}, nil
		}
		return nil, err
	}
	if o == nil {
		return map[string]any{"code": 13}, nil
	}
	if o.Status == "paid" || o.Status == "failed" {
		return map[string]any{"code": 13}, nil
	}
	return map[string]any{"code": 0}, nil
}

func (s *service) handleCPPay(ctx context.Context, payload map[string]any, orderID *int64, subscriptionID string) (map[string]any, error) {
	var order *Order
	if orderID != nil {
		o, err := s.repo.GetById(ctx, *orderID)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return nil, err
			}
		} else {
			order = o
		}
	}

	transactionID := cpExtractString(payload, "TransactionId")
	token := cpExtractString(payload, "Token")
	email := cpExtractString(payload, "Email")
	description := cpExtractString(payload, "Description")
	if description == "" && order != nil {
		if pl, err := s.getPlanByID(ctx, order.PlanId); err == nil {
			description = fmt.Sprintf("План %s", pl.Code)
		}
	}

	paidAt := time.Now()
	if dt := cpExtractString(payload, "DateTime"); dt != "" {
		if t, err := cpParseDateTime(dt); err == nil {
			paidAt = t
		}
	}

	amount, hasAmount := cpExtractFloat(payload, "Amount")

	if order == nil {
		if subscriptionID == "" {
			return nil, fmt.Errorf("order not found for pay notification")
		}
		sub, err := s.subRepo.GetByExternalSubscription(ctx, subscriptionID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, fmt.Errorf("subscription %s not found", subscriptionID)
			}
			return nil, err
		}
		if sub == nil {
			return nil, fmt.Errorf("subscription %s not found", subscriptionID)
		}
		newOrder := Order{
			Status:      "pending",
			Provider:    providerCloudPayments,
			UserId:      sub.UserId,
			PlanId:      sub.PlanId,
			AmountMinor: sub.AmountMinor,
			Currency:    sub.Currency,
			AdCode:      sub.AdCode,
		}
		if hasAmount {
			newOrder.AmountMinor = cpAmountToMinor(amount)
		}
		created, err := s.repo.Create(ctx, newOrder)
		if err != nil {
			return nil, err
		}
		order = created
		orderID = &order.Id
	}

	order.Provider = providerCloudPayments
	if hasAmount && order.AmountMinor == 0 {
		order.AmountMinor = cpAmountToMinor(amount)
	}
	if transactionID != "" {
		order.CpTransId = stringPtr(transactionID)
	}
	if cpOrder := cpExtractString(payload, "OrderId"); cpOrder != "" && order.CpOrderId == nil {
		order.CpOrderId = stringPtr(cpOrder)
	}
	if subscriptionID != "" {
		order.CpSubId = stringPtr(subscriptionID)
	}
	order.LastError = nil

	planData, err := s.getPlanByID(ctx, order.PlanId)
	if err != nil {
		return nil, err
	}
	if description == "" {
		description = fmt.Sprintf("Подписка %s", planData.Code)
	}

	cpSubID := stringPtr(subscriptionID)
	if cpSubID == nil && token != "" {
		subID, err := s.createCloudPaymentsSubscription(ctx, order, planData, token, email, description, paidAt)
		if err != nil {
			return nil, err
		}
		cpSubID = stringPtr(subID)
		order.CpSubId = cpSubID
	}

	if cpSubID == nil && subscriptionID != "" {
		cpSubID = stringPtr(subscriptionID)
	}

	if err := s.processPaidOrder(ctx, order, paidAt, cpSubID); err != nil {
		return nil, err
	}

	return map[string]any{"code": 0}, nil
}

func (s *service) handleCPFail(ctx context.Context, payload map[string]any, orderID *int64, subscriptionID string) (map[string]any, error) {
	reason := cpExtractString(payload, "Reason")
	if orderID != nil {
		o, err := s.repo.GetById(ctx, *orderID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		if err == nil && o != nil {
			o.Status = "failed"
			o.PaidAt = nil
			o.LastError = stringPtr(reason)
			if _, err := s.repo.Update(ctx, *o); err != nil {
				return nil, err
			}
		}
	}

	if subscriptionID != "" {
		sub, err := s.subRepo.GetByExternalSubscription(ctx, subscriptionID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		if err == nil && sub != nil {
			sub.Status = "past_due"
			if _, err := s.subRepo.Update(ctx, *sub); err != nil {
				return nil, err
			}
		}
	}

	return map[string]any{"code": 0}, nil
}

func (s *service) handleCPRecurrent(ctx context.Context, payload map[string]any, subscriptionID string) (map[string]any, error) {
	if subscriptionID == "" {
		return map[string]any{"code": 0}, nil
	}
	sub, err := s.subRepo.GetByExternalSubscription(ctx, subscriptionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return map[string]any{"code": 0}, nil
		}
		return nil, err
	}
	if sub == nil {
		return map[string]any{"code": 0}, nil
	}

	status := strings.ToLower(cpExtractString(payload, "Status"))
	switch status {
	case "active":
		sub.Status = "active"
	case "suspended":
		sub.Status = "past_due"
	case "cancelled", "canceled", "finished", "completed":
		now := time.Now()
		sub.CanceledAt = &now
		sub.CancelAt = nil
		sub.ExternalSubscription = nil // убрать связь с CloudPayments

		// Понизить до START плана
		startPlan, err := s.planRepo.GetByCode(ctx, "START")
		if err == nil && startPlan != nil {
			sub.PlanId = startPlan.Id
			sub.AmountMinor = startPlan.AmountMinor
			sub.Currency = startPlan.Currency
			sub.BillingPeriod = startPlan.BillingPeriod
			sub.Status = "active" // START - бесплатный активный план
			sub.PeriodStart = now
			sub.PeriodEnd = addMonths(now, monthsForBillingPeriod(startPlan.BillingPeriod))
		} else {
			sub.Status = "canceled"
		}
	default:
		sub.Status = "past_due"
	}

	if next := cpExtractString(payload, "NextTransactionDate"); next != "" {
		if t, err := cpParseDateTime(next); err == nil {
			sub.PeriodEnd = t
		}
	}

	if _, err := s.subRepo.Update(ctx, *sub); err != nil {
		return nil, err
	}

	return map[string]any{"code": 0}, nil
}

func (s *service) createCloudPaymentsSubscription(ctx context.Context, order *Order, pl *plan.Plan, token, email, description string, lastPaid time.Time) (string, error) {
	if description == "" {
		description = fmt.Sprintf("Подписка %s", pl.Code)
	}
	nextStart := addMonths(lastPaid, monthsForBillingPeriod(pl.BillingPeriod))
	interval, period := cpSchedule(pl.BillingPeriod)
	receipt := buildCloudPaymentsReceipt(description, email, float64(order.AmountMinor), order.Currency)

	req := CloudPaymentsSubscriptionRequest{
		Token:           token,
		AccountID:       order.UserId,
		Description:     description,
		Email:           email,
		Amount:          float64(order.AmountMinor),
		Currency:        order.Currency,
		RequireConfirm:  false,
		StartDate:       nextStart,
		Interval:        interval,
		Period:          period,
		CustomerReceipt: receipt,
	}

	resp, err := s.cp.CreateSubscription(ctx, req)
	if err != nil {
		return "", err
	}
	return resp.ID, nil
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
