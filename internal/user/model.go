package user

import (
	"time"
)

// User представляет структуру пользователя в системе.
type User struct {
	Id                    string     `json:"id"`
	Username              string     `json:"username"`
	PasswordHash          string     `json:"-"`
	IsAcceptTerms         bool       `json:"-"`
	IsAcceptPrivacy       bool       `json:"-"`
	Is18                  bool       `json:"-"`
	TelegramChatId        *string    `json:"-"`
	TelegramLinkedAt      *time.Time `json:"-"`
	TelegramNotifications bool       `json:"-"`
	TokenVersion          int64      `json:"-"`
	LoginedAt             *time.Time `json:"-"`
	CreatedAt             time.Time  `json:"-"`
	UpdatedAt             time.Time  `json:"-"`
	DeletedAt             *time.Time `json:"-"`
}

// UserCreate представляет структуру для создания пользователя
type UserCreate struct {
	Username     string
	PasswordHash string
}
