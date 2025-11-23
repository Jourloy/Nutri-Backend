-- News and Updates table
-- Stores news and updates with multilingual support
CREATE TABLE IF NOT EXISTS news (
    id BIGSERIAL PRIMARY KEY,

    -- Content (multilingual)
    title_ru TEXT NOT NULL,
    title_en TEXT NOT NULL,
    content_ru TEXT NOT NULL, -- Rich text content (HTML from TipTap)
    content_en TEXT NOT NULL, -- Rich text content (HTML from TipTap)

    -- Media
    image_url TEXT, -- URL to image or GIF

    -- Metadata
    is_published BOOLEAN NOT NULL DEFAULT false,
    published_at TIMESTAMP,
    priority INTEGER NOT NULL DEFAULT 0, -- Higher priority = shown first

    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- Soft delete
    deleted_at TIMESTAMP
);

CREATE INDEX idx_news_published ON news(is_published, published_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_news_priority ON news(priority DESC, created_at DESC) WHERE deleted_at IS NULL AND is_published = true;
CREATE INDEX idx_news_created_at ON news(created_at DESC);

-- Add last viewed news tracking to users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_viewed_news_id BIGINT REFERENCES news(id);
CREATE INDEX IF NOT EXISTS idx_users_last_viewed_news ON users(last_viewed_news_id);

-- Comments for documentation
COMMENT ON TABLE news IS 'Stores news and updates with multilingual support (Russian and English)';
COMMENT ON COLUMN news.content_ru IS 'Rich text content in Russian (HTML from Mantine TipTap editor)';
COMMENT ON COLUMN news.content_en IS 'Rich text content in English (HTML from Mantine TipTap editor)';
COMMENT ON COLUMN news.priority IS 'Display priority - higher values appear first';
COMMENT ON COLUMN users.last_viewed_news_id IS 'ID of the last news item viewed by user';
