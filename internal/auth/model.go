package auth

import "time"

type LoginData struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type RegisterData struct {
	Username                   string `json:"username"`
	Password                   string `json:"password"`
	Locale                     string `json:"locale"`
	Is18                       bool   `json:"is18"`
	AcceptTerms                bool   `json:"acceptTerms"`
	AcceptPrivacy              bool   `json:"acceptPrivacy"`
	PersonalDataConsent        bool   `json:"personalDataConsent"`
	PersonalDataConsentVersion string `json:"personalDataConsentVersion"`
}

// PasswordResetToken represents a password reset token in the database
type PasswordResetToken struct {
	Id        string     `json:"id" db:"id"`
	UserId    string     `json:"-" db:"user_id"`
	Token     string     `json:"token" db:"token"`
	Method    string     `json:"method" db:"method"` // "telegram" or "email"
	ExpiresAt time.Time  `json:"expiresAt" db:"expires_at"`
	UsedAt    *time.Time `json:"usedAt,omitempty" db:"used_at"`
	CreatedAt time.Time  `json:"createdAt" db:"created_at"`
}

// RequestPasswordResetData is the request body for password reset
type RequestPasswordResetData struct {
	Username string `json:"username"`
}

// ResetPasswordData is the request body for setting a new password
type ResetPasswordData struct {
	Token       string `json:"token"`
	NewPassword string `json:"newPassword"`
}

// PasswordResetResponse is returned when requesting a password reset
type PasswordResetResponse struct {
	Method  string `json:"method"`          // "telegram" or "email"
	Message string `json:"message"`         // User-friendly message
	Sent    bool   `json:"sent"`            // Whether the reset link was sent
	Email   string `json:"email,omitempty"` // Masked email if sent via email
}
