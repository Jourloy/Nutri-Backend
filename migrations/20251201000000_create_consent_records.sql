-- Create consent_records table for GDPR compliance
CREATE TABLE IF NOT EXISTS consent_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,  -- nullable for anonymous users
    ip_address TEXT NOT NULL,
    user_agent TEXT,
    consent_given BOOLEAN NOT NULL,
    consent_type TEXT NOT NULL DEFAULT 'analytics',
    consent_date TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX idx_consent_records_user_id ON consent_records(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX idx_consent_records_ip ON consent_records(ip_address, consent_date DESC);
CREATE INDEX idx_consent_records_latest ON consent_records(user_id, consent_date DESC) WHERE user_id IS NOT NULL;
