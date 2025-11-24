package email

import (
	"time"
)

// VerificationCode представляет структуру кода верификации email
type VerificationCode struct {
	Id         string     `json:"id" db:"id"`
	UserId     string     `json:"userId" db:"user_id"`
	Email      string     `json:"email" db:"email"`
	Code       string     `json:"code" db:"code"`
	ExpiresAt  time.Time  `json:"expiresAt" db:"expires_at"`
	VerifiedAt *time.Time `json:"verifiedAt,omitempty" db:"verified_at"`
	CreatedAt  time.Time  `json:"createdAt" db:"created_at"`
}

// VerificationCodeCreate представляет структуру для создания кода верификации
type VerificationCodeCreate struct {
	UserId    string    `db:"user_id"`
	Email     string    `db:"email"`
	Code      string    `db:"code"`
	ExpiresAt time.Time `db:"expires_at"`
}
