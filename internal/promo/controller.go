package promo

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"

	"github.com/charmbracelet/log"
	"github.com/go-chi/chi/v5"

	"github.com/jourloy/nutri02/internal/auth"
)

var (
	logger = log.NewWithOptions(os.Stderr, log.Options{
		Prefix: "[promo]",
		Level:  log.DebugLevel,
	})
)

type Controller struct {
	service Service
}

func NewController() *Controller {
	service := NewService()
	return &Controller{service: service}
}

func (c *Controller) RegisterRoutes(router chi.Router) {
	// Public endpoints
	router.Post("/promo/validate", c.ValidatePromoCode)

	logger.Info("╔═════ Promo")
	logger.Info("║ POST   /promo/validate")
	logger.Info("╚═════")
}

func (c *Controller) RegisterAdminRoutes(router chi.Router) {
	router.Route("/promo", func(r chi.Router) {
		r.Get("/", c.GetAllPromoCodes)
		r.Post("/", c.CreatePromoCode)
		r.Put("/{id}", c.UpdatePromoCode)
		r.Delete("/{id}", c.DeletePromoCode)
	})

	logger.Info("╔═════ Promo Admin")
	logger.Info("║ GET    /admin/promo")
	logger.Info("║ POST   /admin/promo")
	logger.Info("║ PUT    /admin/promo/{id}")
	logger.Info("║ DELETE /admin/promo/{id}")
	logger.Info("╚═════")
}

// ValidatePromoCode проверяет промокод и возвращает расчет скидки
func (c *Controller) ValidatePromoCode(w http.ResponseWriter, r *http.Request) {
	var req ValidatePromoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Code == "" || req.PlanId == 0 || req.AmountMinor == 0 {
		http.Error(w, `{"error": "missing required fields"}`, http.StatusBadRequest)
		return
	}

	response, err := c.service.ValidatePromoCode(r.Context(), &req)
	if err != nil {
		logger.Error("Error validating promo code", "error", err)
		http.Error(w, `{"error": "internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetAllPromoCodes возвращает все промокоды (admin)
func (c *Controller) GetAllPromoCodes(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	if limit <= 0 {
		limit = 50
	}

	promoCodes, err := c.service.GetAllPromoCodes(r.Context(), limit, offset)
	if err != nil {
		logger.Error("Error getting promo codes", "error", err)
		http.Error(w, `{"error": "internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(promoCodes)
}

// CreatePromoCode создает новый промокод (admin)
func (c *Controller) CreatePromoCode(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, `{"error": "unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req PromoCodeCreate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Валидация
	if req.Code == "" || req.DiscountType == "" || req.DiscountValue <= 0 {
		http.Error(w, `{"error": "missing required fields"}`, http.StatusBadRequest)
		return
	}

	if req.DiscountType != "percent" && req.DiscountType != "fixed" {
		http.Error(w, `{"error": "discount_type must be 'percent' or 'fixed'"}`, http.StatusBadRequest)
		return
	}

	promoCode, err := c.service.CreatePromoCode(r.Context(), user.Id, &req)
	if err != nil {
		logger.Error("Error creating promo code", "error", err)
		http.Error(w, `{"error": "failed to create promo code"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(promoCode)
}

// UpdatePromoCode обновляет промокод (admin)
func (c *Controller) UpdatePromoCode(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, `{"error": "invalid promo code id"}`, http.StatusBadRequest)
		return
	}

	var req PromoCodeCreate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	promoCode, err := c.service.UpdatePromoCode(r.Context(), id, &req)
	if err != nil {
		logger.Error("Error updating promo code", "error", err)
		http.Error(w, `{"error": "failed to update promo code"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(promoCode)
}

// DeletePromoCode удаляет промокод (admin)
func (c *Controller) DeletePromoCode(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, `{"error": "invalid promo code id"}`, http.StatusBadRequest)
		return
	}

	if err := c.service.DeletePromoCode(r.Context(), id); err != nil {
		logger.Error("Error deleting promo code", "error", err)
		http.Error(w, `{"error": "failed to delete promo code"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"success": true}`))
}
