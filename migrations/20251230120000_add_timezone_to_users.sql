-- Add timezone field to users table
ALTER TABLE users
ADD COLUMN timezone VARCHAR(64) DEFAULT 'UTC';

COMMENT ON COLUMN users.timezone IS 'IANA timezone name, e.g. Europe/Moscow, America/New_York';
