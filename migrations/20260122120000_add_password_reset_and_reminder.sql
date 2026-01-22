-- Password reset tokens table
CREATE TABLE IF NOT EXISTS password_reset_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token TEXT NOT NULL UNIQUE,
    method TEXT NOT NULL CHECK (method IN ('telegram', 'email')),
    expires_at TIMESTAMP NOT NULL,
    used_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_user_id ON password_reset_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_token ON password_reset_tokens(token);
CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_expires_at ON password_reset_tokens(expires_at);

-- Track email sends per day for rate limiting
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_reset_emails_today INTEGER DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_reset_emails_date DATE;

-- Track last daily reminder sent
ALTER TABLE telegram_profiles ADD COLUMN IF NOT EXISTS last_daily_reminder_at TIMESTAMP;
