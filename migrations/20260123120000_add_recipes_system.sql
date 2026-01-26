-- Recipe Books
CREATE TABLE IF NOT EXISTS recipe_books (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(256) NOT NULL,
    book_type VARCHAR(20) NOT NULL DEFAULT 'user' CHECK (book_type IN ('nutri', 'user')),
    share_token VARCHAR(64) UNIQUE,
    is_shared BOOLEAN NOT NULL DEFAULT FALSE,
    og_image_url TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_recipe_books_user ON recipe_books(user_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_recipe_books_share_token ON recipe_books(share_token) WHERE share_token IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_recipe_books_type ON recipe_books(book_type) WHERE deleted_at IS NULL;

-- Recipe Categories
CREATE TABLE IF NOT EXISTS recipe_categories (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    slug VARCHAR(100) UNIQUE,
    name_ru VARCHAR(255) NOT NULL,
    name_en VARCHAR(255),
    category_type VARCHAR(20) NOT NULL DEFAULT 'user' CHECK (category_type IN ('system', 'user')),
    icon VARCHAR(100),
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_recipe_categories_user ON recipe_categories(user_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_recipe_categories_type ON recipe_categories(category_type) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_recipe_categories_slug ON recipe_categories(slug) WHERE slug IS NOT NULL AND deleted_at IS NULL;

-- Recipe Tags
CREATE TABLE IF NOT EXISTS recipe_tags (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    slug VARCHAR(100) UNIQUE,
    name_ru VARCHAR(100) NOT NULL,
    name_en VARCHAR(100),
    tag_type VARCHAR(20) NOT NULL DEFAULT 'user' CHECK (tag_type IN ('system', 'user')),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_recipe_tags_user ON recipe_tags(user_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_recipe_tags_type ON recipe_tags(tag_type) WHERE deleted_at IS NULL;

-- Recipes
CREATE TABLE IF NOT EXISTS recipes (
    id BIGSERIAL PRIMARY KEY,
    book_id BIGINT NOT NULL REFERENCES recipe_books(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    slug VARCHAR(255) UNIQUE,

    -- Bilingual Content (EN optional for user recipes)
    title_ru VARCHAR(500) NOT NULL,
    title_en VARCHAR(500),
    description_ru TEXT,
    description_en TEXT,

    -- Main Image
    main_image_url TEXT,

    -- Timing (in minutes)
    prep_time INTEGER,
    cook_time INTEGER,
    total_time INTEGER,

    -- Servings
    servings INTEGER NOT NULL DEFAULT 1,
    servings_unit VARCHAR(50),

    -- Nutrition (per serving)
    calories NUMERIC(8, 2),
    protein NUMERIC(8, 2),
    fat NUMERIC(8, 2),
    carbs NUMERIC(8, 2),
    fiber NUMERIC(8, 2),
    nutrition_calculated_by_ai BOOLEAN NOT NULL DEFAULT FALSE,

    -- Difficulty
    difficulty VARCHAR(20) CHECK (difficulty IN ('easy', 'medium', 'hard')),

    -- Sharing
    share_token VARCHAR(64) UNIQUE,
    is_public BOOLEAN NOT NULL DEFAULT FALSE,
    og_image_url TEXT,

    -- SEO (for Nutri recipes)
    meta_description_ru VARCHAR(320),
    meta_description_en VARCHAR(320),

    -- Statistics
    view_count INTEGER NOT NULL DEFAULT 0,
    copy_count INTEGER NOT NULL DEFAULT 0,
    copied_from_id BIGINT REFERENCES recipes(id) ON DELETE SET NULL,

    -- Category
    category_id BIGINT REFERENCES recipe_categories(id) ON DELETE SET NULL,

    -- Publishing
    published_at TIMESTAMP,

    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_recipes_book ON recipes(book_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_recipes_user ON recipes(user_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_recipes_slug ON recipes(slug) WHERE slug IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_recipes_public ON recipes(is_public, created_at DESC) WHERE deleted_at IS NULL AND is_public = TRUE;
CREATE INDEX IF NOT EXISTS idx_recipes_share_token ON recipes(share_token) WHERE share_token IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_recipes_category ON recipes(category_id) WHERE deleted_at IS NULL;

-- Recipe Steps
CREATE TABLE IF NOT EXISTS recipe_steps (
    id BIGSERIAL PRIMARY KEY,
    recipe_id BIGINT NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    step_number INTEGER NOT NULL,
    instruction_ru TEXT NOT NULL,
    instruction_en TEXT,
    image_url TEXT,
    duration_minutes INTEGER,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_recipe_steps_recipe ON recipe_steps(recipe_id, step_number);

-- Recipe Ingredients
CREATE TABLE IF NOT EXISTS recipe_ingredients (
    id BIGSERIAL PRIMARY KEY,
    recipe_id BIGINT NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    name_ru VARCHAR(255) NOT NULL,
    name_en VARCHAR(255),
    amount NUMERIC(10, 2),
    unit VARCHAR(50),
    calories NUMERIC(8, 2),
    protein NUMERIC(8, 2),
    fat NUMERIC(8, 2),
    carbs NUMERIC(8, 2),
    fiber NUMERIC(8, 2),
    is_optional BOOLEAN NOT NULL DEFAULT FALSE,
    group_name VARCHAR(100),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_recipe_ingredients_recipe ON recipe_ingredients(recipe_id, sort_order);

-- Recipe Tags Map (many-to-many)
CREATE TABLE IF NOT EXISTS recipe_tags_map (
    recipe_id BIGINT NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    tag_id BIGINT NOT NULL REFERENCES recipe_tags(id) ON DELETE CASCADE,
    PRIMARY KEY (recipe_id, tag_id)
);

CREATE INDEX IF NOT EXISTS idx_recipe_tags_map_tag ON recipe_tags_map(tag_id);

-- Recipe Images (additional gallery images)
CREATE TABLE IF NOT EXISTS recipe_images (
    id BIGSERIAL PRIMARY KEY,
    recipe_id BIGINT NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    image_url TEXT NOT NULL,
    caption_ru VARCHAR(255),
    caption_en VARCHAR(255),
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_recipe_images_recipe ON recipe_images(recipe_id, sort_order);

-- Seed Nutri Book (only insert if no nutri book exists)
INSERT INTO recipe_books (book_type, name, is_shared)
SELECT 'nutri', 'Nutri Recipes', TRUE
WHERE NOT EXISTS (SELECT 1 FROM recipe_books WHERE book_type = 'nutri');

-- Seed System Categories
INSERT INTO recipe_categories (category_type, slug, name_ru, name_en, icon, sort_order) VALUES
('system', 'breakfast', 'Завтраки', 'Breakfast', '🍳', 1),
('system', 'lunch', 'Обеды', 'Lunch', '🥗', 2),
('system', 'dinner', 'Ужины', 'Dinner', '🍽️', 3),
('system', 'snacks', 'Перекусы', 'Snacks', '🥪', 4),
('system', 'desserts', 'Десерты', 'Desserts', '🍰', 5),
('system', 'drinks', 'Напитки', 'Drinks', '🥤', 6),
('system', 'salads', 'Салаты', 'Salads', '🥗', 7),
('system', 'soups', 'Супы', 'Soups', '🍜', 8),
('system', 'baking', 'Выпечка', 'Baking', '🥐', 9),
('system', 'smoothies', 'Смузи', 'Smoothies', '🥤', 10)
ON CONFLICT (slug) DO NOTHING;

-- Seed System Tags
INSERT INTO recipe_tags (tag_type, slug, name_ru, name_en) VALUES
('system', 'quick', 'Быстрый', 'Quick'),
('system', 'easy', 'Простой', 'Easy'),
('system', 'vegetarian', 'Вегетарианский', 'Vegetarian'),
('system', 'vegan', 'Веганский', 'Vegan'),
('system', 'gluten-free', 'Без глютена', 'Gluten-free'),
('system', 'dairy-free', 'Без молока', 'Dairy-free'),
('system', 'low-carb', 'Низкоуглеводный', 'Low-carb'),
('system', 'high-protein', 'Высокобелковый', 'High-protein'),
('system', 'keto', 'Кето', 'Keto'),
('system', 'budget', 'Бюджетный', 'Budget-friendly')
ON CONFLICT (slug) DO NOTHING;
