package ticket

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"

	"github.com/charmbracelet/log"
	"github.com/go-chi/chi/v5"

	"github.com/jourloy/nutri02/internal/auth"
	"github.com/jourloy/nutri02/internal/consent"
	"github.com/jourloy/nutri02/internal/middlewares"
)

var (
	logger = log.NewWithOptions(os.Stderr, log.Options{
		Prefix: "[ticket]",
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
	router.Route("/tickets", func(r chi.Router) {
		r.With(middlewares.RequireConsent(consent.TypePersonalDataProcessing)).Post("/", c.CreateTicket)
		r.Get("/", c.GetUserTickets)
		r.Get("/{id}", c.GetTicket)
		r.With(middlewares.RequireConsent(consent.TypePersonalDataProcessing)).Post("/{id}/messages", c.AddMessage)
		r.Post("/{id}/close", c.CloseTicket)
	})

	logger.Info("╔═════ Tickets")
	logger.Info("║ POST   /tickets")
	logger.Info("║ GET    /tickets")
	logger.Info("║ GET    /tickets/{id}")
	logger.Info("║ POST   /tickets/{id}/messages")
	logger.Info("║ POST   /tickets/{id}/close")
	logger.Info("╚═════")
}

func (c *Controller) RegisterAdminRoutes(router chi.Router) {
	router.Route("/admin/tickets", func(r chi.Router) {
		r.Get("/", c.GetAllTickets)
		r.Get("/{id}", c.GetTicketAdmin)
		r.Patch("/{id}", c.UpdateTicket)
		r.Post("/{id}/messages", c.AddMessageAdmin)
	})

	logger.Info("╔═════ Tickets Admin")
	logger.Info("║ GET    /admin/tickets")
	logger.Info("║ GET    /admin/tickets/{id}")
	logger.Info("║ PATCH  /admin/tickets/{id}")
	logger.Info("║ POST   /admin/tickets/{id}/messages")
	logger.Info("╚═════")
}

// CreateTicket создает новый тикет
func (c *Controller) CreateTicket(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, `{"error": "unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req CreateTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Subject == "" || req.Message == "" {
		http.Error(w, `{"error": "subject and message are required"}`, http.StatusBadRequest)
		return
	}

	ticket, err := c.service.CreateTicket(r.Context(), user.Id, &req)
	if err != nil {
		logger.Error("Error creating ticket", "error", err)
		http.Error(w, `{"error": "failed to create ticket"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(ticket)
}

// GetUserTickets возвращает тикеты пользователя
func (c *Controller) GetUserTickets(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, `{"error": "unauthorized"}`, http.StatusUnauthorized)
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	if limit <= 0 {
		limit = 50
	}

	tickets, err := c.service.GetUserTickets(r.Context(), user.Id, limit, offset)
	if err != nil {
		logger.Error("Error getting user tickets", "error", err)
		http.Error(w, `{"error": "internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tickets)
}

// GetTicket возвращает тикет с сообщениями
func (c *Controller) GetTicket(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, `{"error": "unauthorized"}`, http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, `{"error": "invalid ticket id"}`, http.StatusBadRequest)
		return
	}

	ticket, err := c.service.GetTicket(r.Context(), id, user.Id, user.IsAdmin)
	if err != nil {
		logger.Error("Error getting ticket", "error", err)
		http.Error(w, `{"error": "internal server error"}`, http.StatusInternalServerError)
		return
	}

	if ticket == nil {
		http.Error(w, `{"error": "ticket not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ticket)
}

// AddMessage добавляет сообщение в тикет
func (c *Controller) AddMessage(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, `{"error": "unauthorized"}`, http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, `{"error": "invalid ticket id"}`, http.StatusBadRequest)
		return
	}

	var req AddMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Message == "" {
		http.Error(w, `{"error": "message is required"}`, http.StatusBadRequest)
		return
	}

	message, err := c.service.AddMessage(r.Context(), id, user.Id, req.Message, false)
	if err != nil {
		logger.Error("Error adding message", "error", err)
		http.Error(w, `{"error": "failed to add message"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(message)
}

// CloseTicket закрывает тикет
func (c *Controller) CloseTicket(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, `{"error": "unauthorized"}`, http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, `{"error": "invalid ticket id"}`, http.StatusBadRequest)
		return
	}

	ticket, err := c.service.GetTicket(r.Context(), id, user.Id, user.IsAdmin)
	if err != nil {
		logger.Error("Error getting ticket", "error", err)
		http.Error(w, `{"error": "internal server error"}`, http.StatusInternalServerError)
		return
	}
	if ticket == nil {
		http.Error(w, `{"error": "ticket not found"}`, http.StatusNotFound)
		return
	}

	if err := c.service.CloseTicket(r.Context(), id); err != nil {
		logger.Error("Error closing ticket", "error", err)
		http.Error(w, `{"error": "failed to close ticket"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"success": true}`))
}

// Admin endpoints

// GetAllTickets возвращает все тикеты (admin)
func (c *Controller) GetAllTickets(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	status := r.URL.Query().Get("status")

	if limit <= 0 {
		limit = 50
	}

	var statusPtr *string
	if status != "" {
		statusPtr = &status
	}

	tickets, err := c.service.GetAllTickets(r.Context(), statusPtr, limit, offset)
	if err != nil {
		logger.Error("Error getting all tickets", "error", err)
		http.Error(w, `{"error": "internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tickets)
}

// GetTicketAdmin возвращает тикет (admin)
func (c *Controller) GetTicketAdmin(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, `{"error": "invalid ticket id"}`, http.StatusBadRequest)
		return
	}

	ticket, err := c.service.GetTicket(r.Context(), id, "", true)
	if err != nil {
		logger.Error("Error getting ticket", "error", err)
		http.Error(w, `{"error": "internal server error"}`, http.StatusInternalServerError)
		return
	}

	if ticket == nil {
		http.Error(w, `{"error": "ticket not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ticket)
}

// UpdateTicket обновляет тикет (admin)
func (c *Controller) UpdateTicket(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, `{"error": "invalid ticket id"}`, http.StatusBadRequest)
		return
	}

	var req UpdateTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	ticket, err := c.service.UpdateTicket(r.Context(), id, &req)
	if err != nil {
		logger.Error("Error updating ticket", "error", err)
		http.Error(w, `{"error": "failed to update ticket"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ticket)
}

// AddMessageAdmin добавляет сообщение от админа
func (c *Controller) AddMessageAdmin(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, `{"error": "unauthorized"}`, http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, `{"error": "invalid ticket id"}`, http.StatusBadRequest)
		return
	}

	var req AddMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Message == "" {
		http.Error(w, `{"error": "message is required"}`, http.StatusBadRequest)
		return
	}

	message, err := c.service.AddMessage(r.Context(), id, user.Id, req.Message, true)
	if err != nil {
		logger.Error("Error adding admin message", "error", err)
		http.Error(w, `{"error": "failed to add message"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(message)
}
