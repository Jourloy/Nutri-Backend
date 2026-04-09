package consent

import (
	"context"

	"github.com/jmoiron/sqlx"

	"github.com/jourloy/somivyn/internal/database"
)

type Repository interface {
	CreateConsentRecord(ctx context.Context, record *ConsentRecord) error
	GetLatestConsentByUserId(ctx context.Context, userId string) (*ConsentRecord, error)
	GetLatestConsentByIP(ctx context.Context, ipAddress string) (*ConsentRecord, error)
}

type repository struct {
	db *sqlx.DB
}

func NewRepository() Repository {
	return &repository{db: database.Database}
}

func (r *repository) CreateConsentRecord(ctx context.Context, record *ConsentRecord) error {
	const q = `
		INSERT INTO consent_records (user_id, ip_address, user_agent, consent_given, consent_type)
		VALUES (:user_id, :ip_address, :user_agent, :consent_given, :consent_type)
		RETURNING id, user_id, ip_address, user_agent, consent_given, consent_type, consent_date, created_at, updated_at;`

	rows, err := r.db.NamedQueryContext(ctx, q, record)
	if err != nil {
		return err
	}
	defer rows.Close()

	if rows.Next() {
		if err := rows.StructScan(record); err != nil {
			return err
		}
	}

	return nil
}

func (r *repository) GetLatestConsentByUserId(ctx context.Context, userId string) (*ConsentRecord, error) {
	const q = `
		SELECT id, user_id, ip_address, user_agent, consent_given, consent_type, consent_date, created_at, updated_at
		FROM consent_records
		WHERE user_id = $1
		ORDER BY consent_date DESC
		LIMIT 1;`

	var record ConsentRecord
	err := r.db.GetContext(ctx, &record, q, userId)
	if err != nil {
		return nil, err
	}

	return &record, nil
}

func (r *repository) GetLatestConsentByIP(ctx context.Context, ipAddress string) (*ConsentRecord, error) {
	const q = `
		SELECT id, user_id, ip_address, user_agent, consent_given, consent_type, consent_date, created_at, updated_at
		FROM consent_records
		WHERE ip_address = $1
		ORDER BY consent_date DESC
		LIMIT 1;`

	var record ConsentRecord
	err := r.db.GetContext(ctx, &record, q, ipAddress)
	if err != nil {
		return nil, err
	}

	return &record, nil
}
