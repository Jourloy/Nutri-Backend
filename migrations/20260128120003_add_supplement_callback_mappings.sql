-- Mapping table for short callback IDs
-- This table maps short sequential IDs to UUID pairs for Telegram button callbacks
-- Reduces callback data from ~91 bytes to ~15 bytes to fit Telegram's 64-byte limit
CREATE TABLE IF NOT EXISTS supplement_callback_mappings (
    id BIGSERIAL PRIMARY KEY,
    supplement_id UUID NOT NULL,
    schedule_id UUID NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL DEFAULT (NOW() + INTERVAL '24 hours')
);

-- Index for faster cleanup and lookup queries
CREATE INDEX IF NOT EXISTS idx_callback_mappings_expires ON supplement_callback_mappings(expires_at);
