-- Achievements categories
CREATE TABLE IF NOT EXISTS achievement_categories (
    id BIGSERIAL PRIMARY KEY,
    key CITEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    position BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Achievements
CREATE TABLE IF NOT EXISTS achievements (
    id BIGSERIAL PRIMARY KEY,
    key CITEXT NOT NULL UNIQUE,
    category_id BIGINT REFERENCES achievement_categories(id) ON DELETE SET NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    icon TEXT,
    color TEXT,
    points INT NOT NULL DEFAULT 0,
    is_secret BOOLEAN NOT NULL DEFAULT FALSE,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    prerequisite_id BIGINT REFERENCES achievements(id) ON DELETE SET NULL,
    criteria JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_achievements_category ON achievements(category_id);
CREATE INDEX IF NOT EXISTS idx_achievements_enabled ON achievements(enabled);

-- User achievements
CREATE TABLE IF NOT EXISTS user_achievements (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    achievement_id BIGINT NOT NULL REFERENCES achievements(id),
    achieved_at TIMESTAMP NOT NULL DEFAULT NOW(),
    progress JSONB NOT NULL DEFAULT '{}'::jsonb
);

ALTER TABLE user_achievements ADD CONSTRAINT user_achievements_uk UNIQUE (user_id, achievement_id);
-- Ensure citext is available for case-insensitive keys
CREATE EXTENSION IF NOT EXISTS "citext";
