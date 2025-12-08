-- Blog Categories
CREATE TABLE IF NOT EXISTS blog_categories (
    id BIGSERIAL PRIMARY KEY,
    slug VARCHAR(100) NOT NULL UNIQUE,
    name_ru VARCHAR(255) NOT NULL,
    name_en VARCHAR(255) NOT NULL,
    description_ru TEXT,
    description_en TEXT,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_blog_categories_slug ON blog_categories(slug) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_blog_categories_sort ON blog_categories(sort_order) WHERE deleted_at IS NULL;

-- Blog Tags
CREATE TABLE IF NOT EXISTS blog_tags (
    id BIGSERIAL PRIMARY KEY,
    slug VARCHAR(100) NOT NULL UNIQUE,
    name_ru VARCHAR(100) NOT NULL,
    name_en VARCHAR(100) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_blog_tags_slug ON blog_tags(slug) WHERE deleted_at IS NULL;

-- Blog Articles
CREATE TABLE IF NOT EXISTS blog_articles (
    id BIGSERIAL PRIMARY KEY,
    slug VARCHAR(255) NOT NULL UNIQUE,

    -- Bilingual Content
    title_ru VARCHAR(500) NOT NULL,
    title_en VARCHAR(500) NOT NULL,
    content_ru TEXT NOT NULL,
    content_en TEXT NOT NULL,

    -- Preview Section
    preview_text_ru TEXT,
    preview_text_en TEXT,
    preview_image_url TEXT,

    -- Category
    category_id BIGINT REFERENCES blog_categories(id) ON DELETE SET NULL,

    -- SEO Fields
    meta_description_ru VARCHAR(320),
    meta_description_en VARCHAR(320),
    og_image_url TEXT,
    canonical_url TEXT,

    -- Access Control: draft, authorized, paid, public
    status VARCHAR(20) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'authorized', 'paid', 'public')),

    -- Metrics (hidden from public)
    view_count INTEGER NOT NULL DEFAULT 0,
    reading_time_minutes INTEGER NOT NULL DEFAULT 1,

    -- Publishing
    published_at TIMESTAMP,

    -- Author
    author_id UUID REFERENCES users(id) ON DELETE SET NULL,

    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_blog_articles_slug ON blog_articles(slug) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_blog_articles_status ON blog_articles(status, published_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_blog_articles_category ON blog_articles(category_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_blog_articles_published ON blog_articles(published_at DESC) WHERE deleted_at IS NULL AND status != 'draft';

-- Blog Article Tags (many-to-many)
CREATE TABLE IF NOT EXISTS blog_article_tags (
    article_id BIGINT NOT NULL REFERENCES blog_articles(id) ON DELETE CASCADE,
    tag_id BIGINT NOT NULL REFERENCES blog_tags(id) ON DELETE CASCADE,
    PRIMARY KEY (article_id, tag_id)
);

CREATE INDEX IF NOT EXISTS idx_blog_article_tags_tag ON blog_article_tags(tag_id);

-- Blog Article Feedback
CREATE TABLE IF NOT EXISTS blog_article_feedback (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    article_id BIGINT NOT NULL REFERENCES blog_articles(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    session_id VARCHAR(255),
    helpful BOOLEAN NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_blog_feedback_article ON blog_article_feedback(article_id);
CREATE INDEX IF NOT EXISTS idx_blog_feedback_stats ON blog_article_feedback(article_id, helpful);

-- Unique constraints for feedback (prevent duplicates)
CREATE UNIQUE INDEX IF NOT EXISTS idx_blog_feedback_user ON blog_article_feedback(article_id, user_id) WHERE user_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_blog_feedback_session ON blog_article_feedback(article_id, session_id) WHERE session_id IS NOT NULL AND user_id IS NULL;

-- Seed initial categories
INSERT INTO blog_categories (slug, name_ru, name_en, description_ru, description_en, sort_order) VALUES
('nutrition-basics', 'Основы питания', 'Nutrition Basics', 'Базовые принципы здорового питания', 'Basic principles of healthy eating', 1),
('recipes', 'Рецепты', 'Recipes', 'Здоровые рецепты для повседневной жизни', 'Healthy recipes for everyday life', 2),
('weight-management', 'Контроль веса', 'Weight Management', 'Советы по управлению весом', 'Weight management tips', 3),
('fitness', 'Фитнес', 'Fitness', 'Тренировки и физическая активность', 'Workouts and physical activity', 4),
('lifestyle', 'Образ жизни', 'Lifestyle', 'Здоровый образ жизни', 'Healthy lifestyle', 5)
ON CONFLICT (slug) DO NOTHING;

-- Seed initial tags
INSERT INTO blog_tags (slug, name_ru, name_en) VALUES
('beginner', 'Для начинающих', 'Beginner'),
('advanced', 'Продвинутый', 'Advanced'),
('quick-tips', 'Быстрые советы', 'Quick Tips'),
('science', 'Наука', 'Science'),
('motivation', 'Мотивация', 'Motivation'),
('meal-prep', 'Подготовка еды', 'Meal Prep'),
('supplements', 'Добавки', 'Supplements')
ON CONFLICT (slug) DO NOTHING;
