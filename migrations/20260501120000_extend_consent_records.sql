ALTER TABLE consent_records
    ADD COLUMN IF NOT EXISTS document_version TEXT NOT NULL DEFAULT '2026-05-01',
    ADD COLUMN IF NOT EXISTS locale TEXT NOT NULL DEFAULT 'ru',
    ADD COLUMN IF NOT EXISTS source TEXT NOT NULL DEFAULT 'web';

UPDATE consent_records
SET consent_type = 'analytics_cookies'
WHERE consent_type = 'analytics';

CREATE INDEX IF NOT EXISTS idx_consent_records_user_type_latest
    ON consent_records(user_id, consent_type, consent_date DESC)
    WHERE user_id IS NOT NULL;
