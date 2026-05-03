package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/jourloy/nutri02/internal/database"
)

type Repository interface {
	CreatePasswordResetToken(ctx context.Context, userId string, method string) (*PasswordResetToken, error)
	GetPasswordResetToken(ctx context.Context, token string) (*PasswordResetToken, error)
	MarkPasswordResetTokenUsed(ctx context.Context, tokenId string) error
	DeleteExpiredPasswordResetTokens(ctx context.Context) error
	GetPasswordResetEmailCountToday(ctx context.Context, userId string) (int, error)
	IncrementPasswordResetEmailCount(ctx context.Context, userId string) error
}

type repository struct {
	db *sqlx.DB
}

func NewRepository() Repository {
	return &repository{db: database.Database}
}

// generateSecureToken generates a cryptographically secure random token
func generateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// CreatePasswordResetToken creates a new password reset token
func (r *repository) CreatePasswordResetToken(ctx context.Context, userId string, method string) (*PasswordResetToken, error) {
	token, err := generateSecureToken(32)
	if err != nil {
		return nil, err
	}

	const q = `
		INSERT INTO password_reset_tokens (user_id, token, method, expires_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, token, method, expires_at, used_at, created_at;`

	expiresAt := time.Now().Add(1 * time.Hour) // Token valid for 1 hour

	var prt PasswordResetToken
	if err := r.db.GetContext(ctx, &prt, q, userId, token, method, expiresAt); err != nil {
		return nil, err
	}
	return &prt, nil
}

// GetPasswordResetToken retrieves a token by its value
func (r *repository) GetPasswordResetToken(ctx context.Context, token string) (*PasswordResetToken, error) {
	const q = `
		SELECT id, user_id, token, method, expires_at, used_at, created_at
		FROM password_reset_tokens
		WHERE token = $1
		LIMIT 1;`

	var prt PasswordResetToken
	if err := r.db.GetContext(ctx, &prt, q, token); err != nil {
		return nil, err
	}
	return &prt, nil
}

// MarkPasswordResetTokenUsed marks a token as used
func (r *repository) MarkPasswordResetTokenUsed(ctx context.Context, tokenId string) error {
	const q = `
		UPDATE password_reset_tokens
		SET used_at = NOW()
		WHERE id = $1;`

	_, err := r.db.ExecContext(ctx, q, tokenId)
	return err
}

// DeleteExpiredPasswordResetTokens cleans up expired tokens
func (r *repository) DeleteExpiredPasswordResetTokens(ctx context.Context) error {
	const q = `DELETE FROM password_reset_tokens WHERE expires_at < NOW() OR used_at IS NOT NULL;`
	_, err := r.db.ExecContext(ctx, q)
	return err
}

// GetPasswordResetEmailCountToday returns count of password reset emails sent today
func (r *repository) GetPasswordResetEmailCountToday(ctx context.Context, userId string) (int, error) {
	const q = `
		SELECT COALESCE(password_reset_emails_today, 0)
		FROM users
		WHERE id = $1 AND password_reset_emails_date = CURRENT_DATE;`

	var count int
	err := r.db.GetContext(ctx, &count, q, userId)
	if err != nil {
		// No row means 0 emails today
		return 0, nil
	}
	return count, nil
}

// IncrementPasswordResetEmailCount increments the email count for today
func (r *repository) IncrementPasswordResetEmailCount(ctx context.Context, userId string) error {
	const q = `
		UPDATE users
		SET password_reset_emails_today = CASE
			WHEN password_reset_emails_date = CURRENT_DATE THEN password_reset_emails_today + 1
			ELSE 1
		END,
		password_reset_emails_date = CURRENT_DATE
		WHERE id = $1;`

	_, err := r.db.ExecContext(ctx, q, userId)
	return err
}
