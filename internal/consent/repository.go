package consent

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"

	"github.com/jourloy/nutri02/internal/database"
)

type Repository interface {
	CreateConsentRecord(ctx context.Context, record *ConsentRecord) error
	GetLatestConsentByUserId(ctx context.Context, userId string) (*ConsentRecord, error)
	GetLatestConsentByUserIdAndType(ctx context.Context, userId string, consentType string) (*ConsentRecord, error)
	GetLatestConsentStatusByUserId(ctx context.Context, userId string) (map[string]ConsentRecord, error)
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
		INSERT INTO consent_records (
			user_id, ip_address, user_agent, consent_given, consent_type,
			document_version, locale, source
		)
		VALUES (
			:user_id, :ip_address, :user_agent, :consent_given, :consent_type,
			:document_version, :locale, :source
		)
		RETURNING id, user_id, ip_address, user_agent, consent_given, consent_type, document_version, locale, source, consent_date, created_at, updated_at;`

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
		SELECT id, user_id, ip_address, user_agent, consent_given, consent_type, document_version, locale, source, consent_date, created_at, updated_at
		FROM consent_records
		WHERE user_id = $1
		ORDER BY consent_date DESC
		LIMIT 1;`

	var record ConsentRecord
	err := r.db.GetContext(ctx, &record, q, userId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &record, nil
}

func (r *repository) GetLatestConsentByUserIdAndType(ctx context.Context, userId string, consentType string) (*ConsentRecord, error) {
	const q = `
		SELECT id, user_id, ip_address, user_agent, consent_given, consent_type, document_version, locale, source, consent_date, created_at, updated_at
		FROM consent_records
		WHERE user_id = $1 AND consent_type = $2
		ORDER BY consent_date DESC
		LIMIT 1;`

	var record ConsentRecord
	err := r.db.GetContext(ctx, &record, q, userId, consentType)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &record, nil
}

func (r *repository) GetLatestConsentStatusByUserId(ctx context.Context, userId string) (map[string]ConsentRecord, error) {
	const q = `
		SELECT DISTINCT ON (consent_type)
			id, user_id, ip_address, user_agent, consent_given, consent_type, document_version, locale, source, consent_date, created_at, updated_at
		FROM consent_records
		WHERE user_id = $1
		ORDER BY consent_type, consent_date DESC;`

	var records []ConsentRecord
	if err := r.db.SelectContext(ctx, &records, q, userId); err != nil {
		return nil, err
	}

	result := make(map[string]ConsentRecord, len(records))
	for _, record := range records {
		result[record.ConsentType] = record
	}

	return result, nil
}

func (r *repository) GetLatestConsentByIP(ctx context.Context, ipAddress string) (*ConsentRecord, error) {
	const q = `
		SELECT id, user_id, ip_address, user_agent, consent_given, consent_type, document_version, locale, source, consent_date, created_at, updated_at
		FROM consent_records
		WHERE ip_address = $1
		ORDER BY consent_date DESC
		LIMIT 1;`

	var record ConsentRecord
	err := r.db.GetContext(ctx, &record, q, ipAddress)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &record, nil
}
