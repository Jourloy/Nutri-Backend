package order

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jourloy/nutri-backend/internal/lib"
)

type CloudPaymentsClient interface {
	CreateOrder(ctx context.Context, req CloudPaymentsOrderRequest) (*CloudPaymentsOrder, error)
	CreateSubscription(ctx context.Context, req CloudPaymentsSubscriptionRequest) (*CloudPaymentsSubscription, error)
}

type cloudPaymentsClient struct {
	baseURL    string
	publicID   string
	secret     string
	httpClient *http.Client
}

func NewCloudPaymentsClient() CloudPaymentsClient {
	base := lib.Config.CloudPaymentsBaseURL
	if base == "" {
		base = "https://api.cloudpayments.ru"
	}
	base = strings.TrimSuffix(base, "/")
	client := &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	return &cloudPaymentsClient{
		baseURL:    base,
		publicID:   lib.Config.CloudPaymentsPublicID,
		secret:     lib.Config.CloudPaymentsAPISecret,
		httpClient: client,
	}
}

type CloudPaymentsOrderRequest struct {
	Amount             float64
	Currency           string
	Description        string
	Email              string
	AccountID          string
	InvoiceID          string
	SuccessRedirectURL string
	FailRedirectURL    string
	SendEmail          bool
	JsonData           map[string]any
}

type CloudPaymentsOrder struct {
	ID  string
	URL string
}

type cloudPaymentsOrderResponse struct {
	Success bool    `json:"Success"`
	Message *string `json:"Message"`
	Model   struct {
		ID  string `json:"Id"`
		URL string `json:"Url"`
	} `json:"Model"`
}

func (c *cloudPaymentsClient) CreateOrder(ctx context.Context, req CloudPaymentsOrderRequest) (*CloudPaymentsOrder, error) {
	payload := map[string]any{
		"Amount":      req.Amount,
		"Currency":    req.Currency,
		"Description": req.Description,
		"AccountId":   req.AccountID,
		"InvoiceId":   req.InvoiceID,
		"SendEmail":   req.SendEmail,
	}
	if req.Email != "" {
		payload["Email"] = req.Email
	}
	if req.SuccessRedirectURL != "" {
		payload["SuccessRedirectUrl"] = req.SuccessRedirectURL
	}
	if req.FailRedirectURL != "" {
		payload["FailRedirectUrl"] = req.FailRedirectURL
	}
	if len(req.JsonData) > 0 {
		payload["JsonData"] = req.JsonData
	}

	var res cloudPaymentsOrderResponse
	if err := c.do(ctx, http.MethodPost, "/orders/create", payload, &res); err != nil {
		return nil, err
	}
	if !res.Success {
		msg := "cloudpayments create order failed"
		if res.Message != nil && *res.Message != "" {
			msg = fmt.Sprintf("%s: %s", msg, *res.Message)
		}
		return nil, fmt.Errorf(msg)
	}
	if res.Model.ID == "" || res.Model.URL == "" {
		return nil, fmt.Errorf("cloudpayments create order returned empty data")
	}
	return &CloudPaymentsOrder{ID: res.Model.ID, URL: res.Model.URL}, nil
}

type CloudPaymentsSubscriptionRequest struct {
	Token           string
	AccountID       string
	Description     string
	Email           string
	Amount          float64
	Currency        string
	StartDate       time.Time
	Interval        string
	Period          int
	RequireConfirm  bool
	CustomerReceipt map[string]any
	MaxPeriods      *int
}

type CloudPaymentsSubscription struct {
	ID string
}

type cloudPaymentsSubscriptionResponse struct {
	Success bool    `json:"Success"`
	Message *string `json:"Message"`
	Model   struct {
		ID string `json:"Id"`
	} `json:"Model"`
}

func (c *cloudPaymentsClient) CreateSubscription(ctx context.Context, req CloudPaymentsSubscriptionRequest) (*CloudPaymentsSubscription, error) {
	payload := map[string]any{
		"Token":               req.Token,
		"AccountId":           req.AccountID,
		"Description":         req.Description,
		"Amount":              req.Amount,
		"Currency":            req.Currency,
		"RequireConfirmation": req.RequireConfirm,
		"StartDate":           req.StartDate.UTC().Format("2006-01-02T15:04:05"),
		"Interval":            req.Interval,
		"Period":              req.Period,
	}
	if req.Email != "" {
		payload["Email"] = req.Email
	}
	if req.CustomerReceipt != nil {
		payload["CustomerReceipt"] = req.CustomerReceipt
	}
	if req.MaxPeriods != nil {
		payload["MaxPeriods"] = *req.MaxPeriods
	}

	var res cloudPaymentsSubscriptionResponse
	if err := c.do(ctx, http.MethodPost, "/subscriptions/create", payload, &res); err != nil {
		return nil, err
	}
	if !res.Success {
		msg := "cloudpayments create subscription failed"
		if res.Message != nil && *res.Message != "" {
			msg = fmt.Sprintf("%s: %s", msg, *res.Message)
		}
		return nil, fmt.Errorf(msg)
	}
	if res.Model.ID == "" {
		return nil, fmt.Errorf("cloudpayments create subscription returned empty id")
	}
	return &CloudPaymentsSubscription{ID: res.Model.ID}, nil
}

func (c *cloudPaymentsClient) do(ctx context.Context, method, path string, payload any, out any) error {
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(raw)
	} else {
		body = nil
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.SetBasicAuth(c.publicID, c.secret)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if out == nil {
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}
		return fmt.Errorf("cloudpayments http %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("cloudpayments http %d", resp.StatusCode)
	}
	return nil
}
