package user

import (
	"time"
)

// User представляет структуру пользователя в системе.
type User struct {
	Id                    string     `json:"id" db:"id"`
	Username              string     `json:"username" db:"username"`
	PasswordHash          string     `json:"-" db:"password_hash"`
	IsAcceptTerms         bool       `json:"-" db:"is_accept_terms"`
	IsAcceptPrivacy       bool       `json:"-" db:"is_accept_privacy"`
	Is18                  bool       `json:"-" db:"is_18"`
	TelegramChatId        *string    `json:"-" db:"telegram_chat_id"`
	TelegramLinkedAt      *time.Time `json:"-" db:"telegram_linked_at"`
	TelegramNotifications bool       `json:"-" db:"telegram_notifications"`
	TokenVersion          int64      `json:"-" db:"token_version"`
	ViewUpdates           int64      `json:"viewUpdates" db:"view_updates"`
	ViewTutorial          int64      `json:"viewTutorial" db:"view_tutorial"`
	LoginedAt             *time.Time `json:"-" db:"logined_at"`
	CreatedAt             time.Time  `json:"-" db:"created_at"`
	UpdatedAt             time.Time  `json:"-" db:"updated_at"`
	DeletedAt             *time.Time `json:"-" db:"deleted_at"`
}

// UserCreate представляет структуру для создания пользователя
type UserCreate struct {
	Username     string `db:"username"`
	PasswordHash string `db:"password_hash"`
}
