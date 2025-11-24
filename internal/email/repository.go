package email

import (
	"context"

	"github.com/jmoiron/sqlx"

	"github.com/jourloy/nutri-backend/internal/database"
)

type Repository interface {
	CreateVerificationCode(ctx context.Context, code *VerificationCodeCreate) (*VerificationCode, error)
	GetVerificationCode(ctx context.Context, userId string, code string) (*VerificationCode, error)
	GetLatestVerificationCode(ctx context.Context, userId string) (*VerificationCode, error)
	MarkAsVerified(ctx context.Context, id string) (*VerificationCode, error)
	DeleteExpiredCodes(ctx context.Context) error
}

type repository struct {
	db *sqlx.DB
}

func NewRepository() Repository {
	return &repository{db: database.Database}
}

// единый список колонок
const verificationCodeColumns = `
    id, user_id, email, code, expires_at, verified_at, created_at
`

func (r *repository) CreateVerificationCode(ctx context.Context, codeCreate *VerificationCodeCreate) (*VerificationCode, error) {
	const insertQ = `
	INSERT INTO email_verification_codes (user_id, email, code, expires_at)
	VALUES (:user_id, :email, :code, :expires_at)
	RETURNING ` + verificationCodeColumns + `;`

	args := map[string]any{
		"user_id":    codeCreate.UserId,
		"email":      codeCreate.Email,
		"code":       codeCreate.Code,
		"expires_at": codeCreate.ExpiresAt,
	}

	rows, err := r.db.NamedQueryContext(ctx, insertQ, args)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var vc VerificationCode
		if err := rows.StructScan(&vc); err != nil {
			return nil, err
		}
		return &vc, nil
	}

	return nil, nil
}

func (r *repository) GetVerificationCode(ctx context.Context, userId string, code string) (*VerificationCode, error) {
	const selectQ = `
	SELECT ` + verificationCodeColumns + `
	FROM email_verification_codes
	WHERE user_id = $1 AND code = $2 AND verified_at IS NULL
	ORDER BY created_at DESC
	LIMIT 1;`

	var vc VerificationCode
	if err := r.db.GetContext(ctx, &vc, selectQ, userId, code); err != nil {
		return nil, err
	}

	return &vc, nil
}

func (r *repository) GetLatestVerificationCode(ctx context.Context, userId string) (*VerificationCode, error) {
	const selectQ = `
	SELECT ` + verificationCodeColumns + `
	FROM email_verification_codes
	WHERE user_id = $1 AND verified_at IS NULL
	ORDER BY created_at DESC
	LIMIT 1;`

	var vc VerificationCode
	if err := r.db.GetContext(ctx, &vc, selectQ, userId); err != nil {
		return nil, err
	}

	return &vc, nil
}

func (r *repository) MarkAsVerified(ctx context.Context, id string) (*VerificationCode, error) {
	const updateQ = `
	UPDATE email_verification_codes
	SET verified_at = NOW()
	WHERE id = $1
	RETURNING ` + verificationCodeColumns + `;`

	var vc VerificationCode
	if err := r.db.GetContext(ctx, &vc, updateQ, id); err != nil {
		return nil, err
	}

	return &vc, nil
}

func (r *repository) DeleteExpiredCodes(ctx context.Context) error {
	const deleteQ = `
	DELETE FROM email_verification_codes
	WHERE expires_at < NOW() AND verified_at IS NULL;`

	_, err := r.db.ExecContext(ctx, deleteQ)
	return err
}
