package order

import (
    "bytes"
    "crypto/sha256"
    "crypto/tls"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "net/http"
    "os"
    "sort"
    "strconv"
    "strings"
    "time"

	"github.com/charmbracelet/log"
	"github.com/jourloy/nutri-backend/internal/lib"
)

type TBankClient interface {
    Init(amountMinor int64, orderId, userId, description, email string, returnURL *string, recursive bool) (paymentURL string, tbOrderId string, err error)
    Charge(rebillId string, amountMinor int64, orderId string) error
}

type tbankClient struct {
	baseURL     string
	terminalKey string
	secret      string
	httpClient  *http.Client
}

func NewTBankClient() TBankClient {
	base := os.Getenv("TBANK_BASE_URL")
	if base == "" {
		base = "https://securepay.tinkoff.ru"
	}
	return &tbankClient{
		baseURL:     base,
		terminalKey: lib.Config.TbankTerminalKey,
		secret:      lib.Config.TbankTerminalPassword,
		httpClient:  &http.Client{Timeout: 15 * time.Second},
	}
}

// signToken builds TBank Token:
// 1) Take all fields that are sent in the request (excluding Token) and add Password=secret
// 2) Sort keys lexicographically
// 3) Concatenate values as strings without separators
// 4) Token = SHA256(hex) of this string
func signToken(secret string, fields map[string]any) string {
	// copy and add Password
	m := make(map[string]string, len(fields)+1)
	for k, v := range fields {
		if k == "Token" || v == nil {
			continue
		}
		switch t := v.(type) {
		case string:
			if t == "" {
				continue
			}
			m[k] = t
		case *string:
			if t != nil && *t != "" {
				m[k] = *t
			}
		case int:
			m[k] = strconv.FormatInt(int64(t), 10)
		case int64:
			m[k] = strconv.FormatInt(t, 10)
		case float64:
			// no floats expected; keep integer-like
			m[k] = strconv.FormatInt(int64(t), 10)
		case bool:
			if t {
				m[k] = "true"
			} else {
				m[k] = "false"
			}
		default:
			// attempt json marshal to string
			b, _ := json.Marshal(v)
			if len(b) > 0 && string(b) != "null" {
				m[k] = string(b)
			}
		}
	}
	m["Password"] = secret
	// sort keys
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	// concat values
	var buf bytes.Buffer
	for _, k := range keys {
		buf.WriteString(m[k])
	}
	sum := sha256.Sum256(buf.Bytes())
	return hex.EncodeToString(sum[:])
}

func (c *tbankClient) Init(amountMinor int64, orderId, userId, description, email string, returnURL *string, recursive bool) (string, string, error) {
    amount := amountMinor * 100
    payload := map[string]any{
        "TerminalKey": c.terminalKey,
        "Amount":      amount,
        "OrderId":     orderId,
        "Description": description,
        "CustomerKey": userId,
    }
    if returnURL != nil {
        payload["SuccessURL"] = *returnURL
    }
    // NotificationURL for RebillId and final status callbacks
    my := lib.Config.MyURL
    if my != "" {
        if !strings.HasPrefix(my, "http://") && !strings.HasPrefix(my, "https://") {
            my = "http://" + my
        }
        payload["NotificationURL"] = fmt.Sprintf("%s/order/notify/tbank", my)
    }
    // Always include FailURL redirecting to frontend with error flag
    front := lib.Config.FrontURL
    if front != "" {
        if !strings.HasPrefix(front, "http://") && !strings.HasPrefix(front, "https://") {
            front = "http://" + front
        }
        payload["FailURL"] = fmt.Sprintf("%s/prices?error=1", front)
    }
	if recursive {
		payload["Recurrent"] = "Y"
	}

	tok := signToken(c.secret, payload)

	{
		logPayload := make(map[string]any, len(payload)+1)
		for k, v := range payload {
			logPayload[k] = v
		}
		logPayload["Token"] = tok
		if b, err := json.Marshal(logPayload); err == nil {
			log.WithPrefix("[tbnk]").Info("Init body", "body", string(b))
		}
	}

    payload["Token"] = tok
	payload["Receipt"] = Receipt{
		Items:    []Item{{Name: description, Quantity: 1, Price: amount, Amount: amount, Tax: "none"}},
		Taxation: "usn_income",
		Email:    email,
	}

	body, _ := json.Marshal(payload)

	c.httpClient.Transport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/v2/Init", c.baseURL), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	var out struct {
		Success    bool   `json:"Success"`
		PaymentURL string `json:"PaymentURL"`
		OrderId    string `json:"OrderId"`
		Message    string `json:"Message"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if !out.Success {
		if out.Message == "" {
			out.Message = fmt.Sprintf("http %d", resp.StatusCode)
		}
		// Full log of TBank response for debugging
		if b, err := json.Marshal(out); err == nil {
			log.WithPrefix("[tbnk]").Error("Init failed", "status", resp.StatusCode, "body", string(b))
		} else {
			log.WithPrefix("[tbnk]").Error("Init failed", "status", resp.StatusCode, "out", out)
		}
		return "", "", fmt.Errorf("tbank init failed: %s", out.Message)
	}

    return out.PaymentURL, out.OrderId, nil
}


func (c *tbankClient) Charge(rebillId string, amountMinor int64, orderId string) error {
	amount := amountMinor * 100
	payload := map[string]any{
		"TerminalKey": c.terminalKey,
		"RebillId":    rebillId,
		"Amount":      amount,
		"OrderId":     orderId,
	}
	// add token
	payload["Token"] = signToken(c.secret, payload)
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/v2/Charge", c.baseURL), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var out struct {
		Success bool   `json:"Success"`
		Message string `json:"Message"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if !out.Success {
		if out.Message == "" {
			out.Message = fmt.Sprintf("http %d", resp.StatusCode)
		}
		return fmt.Errorf("tbank charge failed: %s", out.Message)
	}
	return nil
}
