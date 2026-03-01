-- Add structured sources storage for blog articles.
ALTER TABLE blog_articles
    ADD COLUMN IF NOT EXISTS sources TEXT[];
