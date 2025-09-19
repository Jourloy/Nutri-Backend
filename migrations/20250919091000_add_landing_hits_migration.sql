-- Create table to store landing (banner) hits by code
CREATE TABLE IF NOT EXISTS landing_hits (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(100) NOT NULL,
    ip VARCHAR(64),
    user_agent TEXT,
    referer TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Index to speed up aggregations by code and date
CREATE INDEX IF NOT EXISTS idx_landing_hits_code_created_at ON landing_hits (code, created_at);
