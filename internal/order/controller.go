package order

import (
    "context"
    "encoding/json"
    "io"
    "net/http"
    "os"
    "strconv"
    "strings"

    "github.com/charmbracelet/log"
    "github.com/go-chi/chi/v5"

    "github.com/jourloy/nutri-backend/internal/auth"
    "github.com/jourloy/nutri-backend/internal/lib"
)

var (
	logger = log.NewWithOptions(os.Stderr, log.Options{Prefix: "[orde]", Level: log.DebugLevel})
)

type Controller struct{ service Service }

func NewController() *Controller { return &Controller{service: NewService()} }

func (c *Controller) RegisterRoutes(r chi.Router) {
    r.Route("/order", func(r chi.Router) {
        r.Post("/init", c.Init)
        r.Get("/paid", c.Paid)
        r.Post("/notify/tbank", c.NotifyTBank)
        r.Get("/all", c.GetAll)
        r.Delete("/{id}", c.Delete)
        r.Post("/ensure-start", c.EnsureStart)
    })
    logger.Info("╔═════ Order")
    logger.Info("║   GET /all")
    logger.Info("║  POST /init")
    logger.Info("║   GET /paid")
    logger.Info("║  POST /notify/tbank")
    logger.Info("║  POST /ensure-start")
    logger.Info("║ DELETE /{id}")
    logger.Info("╚═════")
}

func (c *Controller) Init(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var p InitPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if p.PlanId <= 0 {
		http.Error(w, "invalid planId", http.StatusBadRequest)
		return
	}

	res, err := c.service.Init(context.Background(), u.Id, p.PlanId, p.Email, p.ReturnURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(res)
}

func (c *Controller) Paid(w http.ResponseWriter, r *http.Request) {
	// Public endpoint: verifies payment and redirects to frontend
	oidStr := r.URL.Query().Get("oid")
	if oidStr == "" {
		http.Error(w, "missing oid", http.StatusBadRequest)
		return
	}
	oid, err := strconv.ParseInt(oidStr, 10, 64)
	if err != nil || oid <= 0 {
		http.Error(w, "invalid oid", http.StatusBadRequest)
		return
	}
	ok, err := c.service.FinalizeReturn(context.Background(), oid)
	// Compose redirect URL
    front := lib.Config.FrontURL
    if front == "" {
        front = "127.0.0.1" // fallback to avoid empty
    }
	if !strings.HasPrefix(front, "http://") && !strings.HasPrefix(front, "https://") {
		front = "http://" + front
	}
	var dest string
	if err != nil || !ok {
		dest = front + "/prices?error=1"
		logger.Error(err)
	} else {
		dest = front + "/app"
	}
    http.Redirect(w, r, dest, http.StatusFound)
}

// NotifyTBank receives asynchronous notifications from TBank (NotificationURL)
// and updates order/subscription, including saving RebillId for recurring charges.
func (c *Controller) NotifyTBank(w http.ResponseWriter, r *http.Request) {
    // Read raw body for signature verification
    raw, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    // Parse into generic map to compute Token
    var mm map[string]any
    if err := json.Unmarshal(raw, &mm); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    recvToken, _ := mm["Token"].(string)
    if recvToken == "" {
        http.Error(w, "no token", http.StatusForbidden)
        return
    }
    // Compute expected token using terminal password
    secret := lib.Config.TbankTerminalPassword
    exp := signToken(secret, mm)
    if strings.ToLower(recvToken) != strings.ToLower(exp) {
        http.Error(w, "invalid token", http.StatusForbidden)
        return
    }
    // Decode to typed payload
    var p TBankWebhook
    if err := json.Unmarshal(raw, &p); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    if err := c.service.HandleTBankWebhook(context.Background(), p); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    w.WriteHeader(http.StatusOK)
}

func (c *Controller) GetAll(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	userID := u.Id
	if u.IsAdmin {
		if q := r.URL.Query().Get("userId"); q != "" {
			userID = q
		} else {
			userID = ""
		}
	}
	res, err := c.service.List(context.Background(), userID, u.IsAdmin)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(res)
}

func (c *Controller) Delete(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	userID := u.Id
	if u.IsAdmin {
		if q := r.URL.Query().Get("userId"); q != "" {
			userID = q
		}
	}
	if err := c.service.Delete(context.Background(), id, userID, u.IsAdmin); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (c *Controller) EnsureStart(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	_, created, err := c.service.EnsureStart(context.Background(), u.Id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{"created": created})
}
