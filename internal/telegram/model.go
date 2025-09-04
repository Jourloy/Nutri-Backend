package telegram

import "time"

// TelegramProfile mirrors telegram_profiles table.
type TelegramProfile struct {
    Id               int64      `json:"id" db:"id"`
    Token            string     `json:"token" db:"token"`
    TelegramId       *string    `json:"telegramId" db:"telegram_id"`
    TelegramUsername *string    `json:"telegramUsername" db:"telegram_username"`
    TelegramAvatar   *string    `json:"telegramAvatar" db:"telegram_avatar"`
    NotifyDaily      bool       `json:"notifyDaily" db:"notify_daily"`
    NotifyStory      bool       `json:"notifyStory" db:"notify_story"`
    UserId           string     `json:"-" db:"user_id"`
    ConnectedAt      *time.Time `json:"connectedAt,omitempty" db:"connected_at"`
    CreatedAt        time.Time  `json:"-" db:"created_at"`
    UpdatedAt        time.Time  `json:"-" db:"updated_at"`
}

// TelegramPublic is a lightweight view returned for public consumption.
type TelegramPublic struct {
    TelegramId       *string `json:"id" db:"telegram_id"`
    TelegramUsername *string `json:"username" db:"telegram_username"`
    TelegramAvatar   *string `json:"avatar" db:"telegram_avatar"`
}

// LinkRequest is used to link a Telegram account by token.
type LinkRequest struct {
    Token            string  `json:"token"`
    TelegramId       *string `json:"telegramId"`
    TelegramUsername *string `json:"telegramUsername"`
    TelegramAvatar   *string `json:"telegramAvatar"`
}

// NotifyUpdate allows partial update of notify flags.
type NotifyUpdate struct {
    NotifyDaily *bool `json:"notifyDaily"`
    NotifyStory *bool `json:"notifyStory"`
}
