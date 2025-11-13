package ticket

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"

	"github.com/jourloy/nutri-backend/internal/database"
)

type Repository interface {
	CreateTicket(ctx context.Context, userId, subject, message, priority string, category *string) (*Ticket, error)
	GetTicket(ctx context.Context, id int64) (*Ticket, error)
	GetUserTickets(ctx context.Context, userId string, limit, offset int) ([]Ticket, error)
	GetAllTickets(ctx context.Context, status *string, limit, offset int) ([]Ticket, error)
	UpdateTicket(ctx context.Context, id int64, req *UpdateTicketRequest) (*Ticket, error)
	AddMessage(ctx context.Context, ticketId int64, userId, message string, isAdmin bool) (*TicketMessage, error)
	GetMessages(ctx context.Context, ticketId int64) ([]TicketMessage, error)
	GetTicketWithMessages(ctx context.Context, id int64) (*TicketWithMessages, error)
	CloseTicket(ctx context.Context, id int64) error
}

type repository struct {
	db *sqlx.DB
}

func NewRepository() Repository {
	return &repository{db: database.Database}
}

const ticketColumns = `
	id, user_id, subject, status, priority, category,
	assigned_to, created_at, updated_at, closed_at
`

const messageColumns = `
	id, ticket_id, user_id, message, is_admin, attachments, created_at
`

func (r *repository) CreateTicket(ctx context.Context, userId, subject, message, priority string, category *string) (*Ticket, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if priority == "" {
		priority = "normal"
	}

	// Создаем тикет
	const ticketQ = `
		INSERT INTO tickets (user_id, subject, priority, category)
		VALUES ($1, $2, $3, $4)
		RETURNING ` + ticketColumns

	var ticket Ticket
	if err := tx.GetContext(ctx, &ticket, ticketQ, userId, subject, priority, category); err != nil {
		return nil, err
	}

	// Добавляем первое сообщение
	const messageQ = `
		INSERT INTO ticket_messages (ticket_id, user_id, message, is_admin)
		VALUES ($1, $2, $3, FALSE)
	`
	if _, err := tx.ExecContext(ctx, messageQ, ticket.Id, userId, message); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &ticket, nil
}

func (r *repository) GetTicket(ctx context.Context, id int64) (*Ticket, error) {
	const q = `SELECT ` + ticketColumns + ` FROM tickets WHERE id = $1`

	var ticket Ticket
	if err := r.db.GetContext(ctx, &ticket, q, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &ticket, nil
}

func (r *repository) GetUserTickets(ctx context.Context, userId string, limit, offset int) ([]Ticket, error) {
	const q = `
		SELECT ` + ticketColumns + `
		FROM tickets
		WHERE user_id = $1
		ORDER BY updated_at DESC
		LIMIT $2 OFFSET $3
	`

	var tickets []Ticket
	if err := r.db.SelectContext(ctx, &tickets, q, userId, limit, offset); err != nil {
		return nil, err
	}
	return tickets, nil
}

func (r *repository) GetAllTickets(ctx context.Context, status *string, limit, offset int) ([]Ticket, error) {
	var q string
	var args []interface{}

	if status != nil {
		q = `
			SELECT ` + ticketColumns + `
			FROM tickets
			WHERE status = $1
			ORDER BY updated_at DESC
			LIMIT $2 OFFSET $3
		`
		args = []interface{}{*status, limit, offset}
	} else {
		q = `
			SELECT ` + ticketColumns + `
			FROM tickets
			ORDER BY updated_at DESC
			LIMIT $1 OFFSET $2
		`
		args = []interface{}{limit, offset}
	}

	var tickets []Ticket
	if err := r.db.SelectContext(ctx, &tickets, q, args...); err != nil {
		return nil, err
	}
	return tickets, nil
}

func (r *repository) UpdateTicket(ctx context.Context, id int64, req *UpdateTicketRequest) (*Ticket, error) {
	// Динамически строим запрос на обновление только переданных полей
	query := `UPDATE tickets SET updated_at = NOW()`
	args := []interface{}{}
	argCount := 1

	if req.Status != nil {
		query += `, status = $` + string(rune('0'+argCount))
		args = append(args, *req.Status)
		argCount++

		// Если статус = closed, устанавливаем closed_at
		if *req.Status == "closed" {
			query += `, closed_at = NOW()`
		}
	}

	if req.Priority != nil {
		query += `, priority = $` + string(rune('0'+argCount))
		args = append(args, *req.Priority)
		argCount++
	}

	if req.AssignedTo != nil {
		query += `, assigned_to = $` + string(rune('0'+argCount))
		args = append(args, *req.AssignedTo)
		argCount++
	}

	query += ` WHERE id = $` + string(rune('0'+argCount))
	args = append(args, id)

	query += ` RETURNING ` + ticketColumns

	var ticket Ticket
	if err := r.db.GetContext(ctx, &ticket, query, args...); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &ticket, nil
}

func (r *repository) AddMessage(ctx context.Context, ticketId int64, userId, message string, isAdmin bool) (*TicketMessage, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Добавляем сообщение
	const msgQ = `
		INSERT INTO ticket_messages (ticket_id, user_id, message, is_admin)
		VALUES ($1, $2, $3, $4)
		RETURNING ` + messageColumns

	var msg TicketMessage
	if err := tx.GetContext(ctx, &msg, msgQ, ticketId, userId, message, isAdmin); err != nil {
		return nil, err
	}

	// Обновляем updated_at тикета
	const updateQ = `UPDATE tickets SET updated_at = NOW() WHERE id = $1`
	if _, err := tx.ExecContext(ctx, updateQ, ticketId); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &msg, nil
}

func (r *repository) GetMessages(ctx context.Context, ticketId int64) ([]TicketMessage, error) {
	const q = `
		SELECT ` + messageColumns + `
		FROM ticket_messages
		WHERE ticket_id = $1
		ORDER BY created_at ASC
	`

	var messages []TicketMessage
	if err := r.db.SelectContext(ctx, &messages, q, ticketId); err != nil {
		return nil, err
	}
	return messages, nil
}

func (r *repository) GetTicketWithMessages(ctx context.Context, id int64) (*TicketWithMessages, error) {
	ticket, err := r.GetTicket(ctx, id)
	if err != nil || ticket == nil {
		return nil, err
	}

	messages, err := r.GetMessages(ctx, id)
	if err != nil {
		return nil, err
	}

	return &TicketWithMessages{
		Ticket:   *ticket,
		Messages: messages,
	}, nil
}

func (r *repository) CloseTicket(ctx context.Context, id int64) error {
	const q = `
		UPDATE tickets
		SET status = 'closed', closed_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, q, id)
	return err
}
