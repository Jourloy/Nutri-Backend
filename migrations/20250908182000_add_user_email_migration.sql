-- Add email column to users for receipts and service communications
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS email CITEXT;

-- Optional index to speed up lookups by email (no uniqueness enforced)
CREATE INDEX IF NOT EXISTS users_email_idx ON users (email);

