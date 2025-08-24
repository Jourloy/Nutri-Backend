package orders

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
)

type Service interface {
	Create(ctx context.Context, uid string, req OrderCreateRequest) (*OrderResponse, error)
}

type service struct {
	repo Repository
}

func NewService() Service {
	return &service{repo: NewRepository()}
}

func (s *service) Create(ctx context.Context, uid string, req OrderCreateRequest) (*OrderResponse, error) {
	p, err := s.repo.FindPlan(ctx, req.Name, req.BillingPeriod)
	if err != nil {
		return nil, err
	}

	order, err := s.repo.Create(ctx, OrderCreate{
		UserId: uid, PlanId: p.Id, AmountMinor: p.AmountMinor, Currency: p.Currency, Status: "NEW",
	})
	if err != nil {
		return nil, err
	}

	terminalKey := os.Getenv("TBANK_TERMINAL_KEY")
	secret := os.Getenv("TBANK_SECRET")
	if terminalKey == "" || secret == "" {
		return nil, errors.New("payment config not set")
	}

	amount := p.AmountMinor * 100
	orderId := strconv.FormatInt(order.Id, 10)

	params := map[string]interface{}{
		"TerminalKey": terminalKey,
		"Amount":      amount,
		"OrderId":     orderId,
		"Description": p.Name,
	}
	params["Token"] = generateToken(params, secret)

	body, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post("https://securepay.tinkoff.ru/v2/Init", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var tResp struct {
		Success    bool   `json:"Success"`
		Status     string `json:"Status"`
		PaymentId  string `json:"PaymentId"`
		PaymentURL string `json:"PaymentURL"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tResp); err != nil {
		return nil, err
	}
	if !tResp.Success {
		return nil, fmt.Errorf("payment init failed: %s", tResp.Status)
	}

	if err := s.repo.UpdatePayment(ctx, order.Id, tResp.PaymentId, tResp.PaymentURL, tResp.Status); err != nil {
		return nil, err
	}

	return &OrderResponse{PaymentURL: tResp.PaymentURL}, nil
}

func generateToken(params map[string]interface{}, password string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		if k == "Token" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		v := params[k]
		switch val := v.(type) {
		case string:
			b.WriteString(val)
		case int:
			b.WriteString(strconv.Itoa(val))
		case int64:
			b.WriteString(strconv.FormatInt(val, 10))
		case float64:
			b.WriteString(strconv.FormatFloat(val, 'f', -1, 64))
		}
	}
	b.WriteString(password)
	sum := sha256.Sum256([]byte(b.String()))
	return strings.ToLower(hex.EncodeToString(sum[:]))
}
