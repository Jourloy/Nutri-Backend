CREATE TABLE IF NOT EXISTS translations (
    id BIGSERIAL PRIMARY KEY,
    namespace VARCHAR(64) NOT NULL,
    translation_key TEXT NOT NULL,
    locale VARCHAR(16) NOT NULL,
    value TEXT NOT NULL,
    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITHOUT TIME ZONE,
    CONSTRAINT translations_locale_check CHECK (locale ~ '^[a-z]{2}(-[A-Za-z]{2})?$'),
    CONSTRAINT translations_unique UNIQUE (namespace, translation_key, locale)
);

CREATE INDEX IF NOT EXISTS idx_translations_locale ON translations (locale) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_translations_namespace ON translations (namespace) WHERE deleted_at IS NULL;
