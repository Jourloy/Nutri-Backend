package ticket

import "context"

type Service interface {
	CreateTicket(ctx context.Context, userId string, req *CreateTicketRequest) (*Ticket, error)
	GetTicket(ctx context.Context, id int64, userId string, isAdmin bool) (*TicketWithMessages, error)
	GetUserTickets(ctx context.Context, userId string, limit, offset int) ([]Ticket, error)
	GetAllTickets(ctx context.Context, status *string, limit, offset int) ([]Ticket, error)
	UpdateTicket(ctx context.Context, id int64, req *UpdateTicketRequest) (*Ticket, error)
	AddMessage(ctx context.Context, ticketId int64, userId string, message string, isAdmin bool) (*TicketMessage, error)
	CloseTicket(ctx context.Context, id int64) error
}

type service struct {
	repo Repository
}

func NewService() Service {
	return &service{repo: NewRepository()}
}

func (s *service) CreateTicket(ctx context.Context, userId string, req *CreateTicketRequest) (*Ticket, error) {
	priority := req.Priority
	if priority == "" {
		priority = "normal"
	}

	return s.repo.CreateTicket(ctx, userId, req.Subject, req.Message, priority, req.Category)
}

func (s *service) GetTicket(ctx context.Context, id int64, userId string, isAdmin bool) (*TicketWithMessages, error) {
	ticketWithMsg, err := s.repo.GetTicketWithMessages(ctx, id)
	if err != nil || ticketWithMsg == nil {
		return nil, err
	}

	// Проверка доступа: пользователь может видеть только свои тикеты, админ - все
	if !isAdmin && ticketWithMsg.Ticket.UserId != userId {
		return nil, nil
	}

	return ticketWithMsg, nil
}

func (s *service) GetUserTickets(ctx context.Context, userId string, limit, offset int) ([]Ticket, error) {
	return s.repo.GetUserTickets(ctx, userId, limit, offset)
}

func (s *service) GetAllTickets(ctx context.Context, status *string, limit, offset int) ([]Ticket, error) {
	return s.repo.GetAllTickets(ctx, status, limit, offset)
}

func (s *service) UpdateTicket(ctx context.Context, id int64, req *UpdateTicketRequest) (*Ticket, error) {
	return s.repo.UpdateTicket(ctx, id, req)
}

func (s *service) AddMessage(ctx context.Context, ticketId int64, userId string, message string, isAdmin bool) (*TicketMessage, error) {
	ticket, err := s.repo.GetTicket(ctx, ticketId)
	if err == nil && ticket != nil {
		var newStatus string

		if !isAdmin {
			// Если пользователь отвечает, и тикет был в статусе "waiting_response",
			// автоматически переводим его в "in_progress"
			if ticket.Status == "waiting_response" {
				newStatus = "in_progress"
			}
		} else {
			// Если админ отвечает, и тикет был в статусе "open" или "in_progress",
			// автоматически переводим его в "waiting_response"
			if ticket.Status == "open" || ticket.Status == "in_progress" {
				newStatus = "waiting_response"
			}
		}

		// Обновляем статус если он изменился
		if newStatus != "" {
			updateReq := &UpdateTicketRequest{
				Status: &newStatus,
			}
			_, _ = s.repo.UpdateTicket(ctx, ticketId, updateReq)
		}
	}

	return s.repo.AddMessage(ctx, ticketId, userId, message, isAdmin)
}

func (s *service) CloseTicket(ctx context.Context, id int64) error {
	return s.repo.CloseTicket(ctx, id)
}
