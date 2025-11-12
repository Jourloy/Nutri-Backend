-- AI Analysis Logs table
-- Stores all AI requests and responses for tracking and cost calculation
CREATE TABLE IF NOT EXISTS ai_analysis_logs (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),

    -- Request data
    request_type TEXT NOT NULL, -- 'food_analysis', 'chat', 'advice', etc.
    image_url TEXT, -- Minio URL to the image
    user_prompt TEXT, -- User's description/question
    total_weight NUMERIC(6,1), -- For food analysis: total weight in grams

    -- Response data
    response_data JSONB, -- Full AI response
    parsed_result JSONB, -- Parsed structured data (calories, protein, etc.)

    -- Cost tracking
    model_used TEXT NOT NULL, -- e.g., 'gpt-4-vision-preview'
    tokens_prompt INTEGER,
    tokens_completion INTEGER,
    estimated_cost_usd NUMERIC(10,6), -- Cost in USD

    -- Status and metadata
    status TEXT NOT NULL DEFAULT 'pending', -- 'pending', 'success', 'error', 'moderated'
    error_message TEXT,
    processing_time_ms INTEGER,

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ai_logs_user_id ON ai_analysis_logs(user_id);
CREATE INDEX idx_ai_logs_created_at ON ai_analysis_logs(created_at);
CREATE INDEX idx_ai_logs_request_type ON ai_analysis_logs(request_type);
CREATE INDEX idx_ai_logs_status ON ai_analysis_logs(status);

-- AI User Limits table
-- Tracks usage limits per user per day
CREATE TABLE IF NOT EXISTS ai_user_limits (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),

    -- Limit configuration
    limit_date DATE NOT NULL DEFAULT CURRENT_DATE,
    request_type TEXT NOT NULL, -- 'food_analysis', 'chat', etc.

    -- Usage tracking
    requests_count INTEGER NOT NULL DEFAULT 0,
    max_requests INTEGER NOT NULL DEFAULT 10, -- Configurable limit

    -- Subscription-based limits
    subscription_tier TEXT, -- 'free', 'premium', etc.

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    UNIQUE(user_id, limit_date, request_type)
);

CREATE INDEX idx_ai_limits_user_date ON ai_user_limits(user_id, limit_date);
CREATE INDEX idx_ai_limits_date ON ai_user_limits(limit_date);

-- AI Violations table
-- Stores violations and inappropriate content
CREATE TABLE IF NOT EXISTS ai_violations (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    analysis_log_id BIGINT REFERENCES ai_analysis_logs(id),

    -- Violation details
    violation_type TEXT NOT NULL, -- 'off_topic', 'inappropriate', 'spam', etc.
    violation_reason TEXT NOT NULL,
    image_url TEXT, -- URL to the violating image
    user_prompt TEXT,

    -- Moderation action
    action_taken TEXT NOT NULL, -- 'warning', 'temp_ban', 'permanent_ban'
    ban_until TIMESTAMP, -- For temporary bans
    reviewed BOOLEAN NOT NULL DEFAULT FALSE,
    reviewed_by UUID REFERENCES users(id),
    reviewed_at TIMESTAMP,

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ai_violations_user_id ON ai_violations(user_id);
CREATE INDEX idx_ai_violations_created_at ON ai_violations(created_at);
CREATE INDEX idx_ai_violations_reviewed ON ai_violations(reviewed);

-- Admin Notifications table
-- General notifications for admin panel
CREATE TABLE IF NOT EXISTS admin_notifications (
    id BIGSERIAL PRIMARY KEY,

    -- Notification details
    notification_type TEXT NOT NULL, -- 'ai_violation', 'user_report', 'system_alert', etc.
    title TEXT NOT NULL,
    message TEXT NOT NULL,
    severity TEXT NOT NULL DEFAULT 'info', -- 'info', 'warning', 'critical'

    -- Related entities
    user_id UUID REFERENCES users(id),
    related_id BIGINT, -- Generic ID for related entity (violation_id, etc.)
    metadata JSONB, -- Additional data

    -- Status
    read BOOLEAN NOT NULL DEFAULT FALSE,
    read_by UUID REFERENCES users(id),
    read_at TIMESTAMP,

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_admin_notif_type ON admin_notifications(notification_type);
CREATE INDEX idx_admin_notif_read ON admin_notifications(read);
CREATE INDEX idx_admin_notif_severity ON admin_notifications(severity);
CREATE INDEX idx_admin_notif_created_at ON admin_notifications(created_at);

-- Add AI ban status to users (or we can check violations table)
ALTER TABLE users ADD COLUMN IF NOT EXISTS ai_banned_until TIMESTAMP;
CREATE INDEX IF NOT EXISTS idx_users_ai_banned ON users(ai_banned_until);
