-- Add email_verified column to users table
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS email_verified BOOLEAN DEFAULT FALSE;

-- Create email_verification_codes table for storing verification codes
CREATE TABLE IF NOT EXISTS email_verification_codes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email CITEXT NOT NULL,
    code TEXT NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    verified_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Index for faster lookups by user_id and code
CREATE INDEX IF NOT EXISTS email_verification_codes_user_id_idx ON email_verification_codes (user_id);
CREATE INDEX IF NOT EXISTS email_verification_codes_code_idx ON email_verification_codes (code);

-- Index for cleanup of expired codes
CREATE INDEX IF NOT EXISTS email_verification_codes_expires_at_idx ON email_verification_codes (expires_at);
