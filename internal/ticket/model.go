package ticket

import (
	"time"

	"github.com/lib/pq"
)

// Ticket представляет обращение пользователя
type Ticket struct {
	Id         int64      `json:"id" db:"id"`
	UserId     string     `json:"user_id" db:"user_id"`
	Subject    string     `json:"subject" db:"subject"`
	Status     string     `json:"status" db:"status"`               // open, in_progress, waiting_response, resolved, closed
	Priority   string     `json:"priority" db:"priority"`           // low, normal, high, urgent
	Category   *string    `json:"category,omitempty" db:"category"` // technical, billing, feature_request, other
	AssignedTo *string    `json:"assigned_to,omitempty" db:"assigned_to"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at" db:"updated_at"`
	ClosedAt   *time.Time `json:"closed_at,omitempty" db:"closed_at"`
}

// TicketMessage представляет сообщение в тикете
type TicketMessage struct {
	Id          int64          `json:"id" db:"id"`
	TicketId    int64          `json:"ticket_id" db:"ticket_id"`
	UserId      string         `json:"user_id" db:"user_id"`
	Message     string         `json:"message" db:"message"`
	IsAdmin     bool           `json:"is_admin" db:"is_admin"`
	Attachments pq.StringArray `json:"attachments,omitempty" db:"attachments"`
	CreatedAt   time.Time      `json:"created_at" db:"created_at"`
}

// TicketWithMessages представляет тикет со всеми сообщениями
type TicketWithMessages struct {
	Ticket   Ticket          `json:"ticket"`
	Messages []TicketMessage `json:"messages"`
}

// CreateTicketRequest представляет запрос на создание тикета
type CreateTicketRequest struct {
	Subject  string  `json:"subject"`
	Message  string  `json:"message"`
	Priority string  `json:"priority,omitempty"`
	Category *string `json:"category,omitempty"`
}

// AddMessageRequest представляет запрос на добавление сообщения
type AddMessageRequest struct {
	Message string `json:"message"`
}

// UpdateTicketRequest представляет запрос на обновление тикета
type UpdateTicketRequest struct {
	Status     *string `json:"status,omitempty"`
	Priority   *string `json:"priority,omitempty"`
	AssignedTo *string `json:"assigned_to,omitempty"`
}
