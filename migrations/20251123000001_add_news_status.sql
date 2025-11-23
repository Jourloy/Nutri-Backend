-- Add status column to news table
ALTER TABLE news ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'draft';

-- Migrate existing data: if is_published = true, set status to 'published', else 'draft'
UPDATE news SET status = CASE WHEN is_published = true THEN 'published' ELSE 'draft' END;

-- Create index for status
CREATE INDEX IF NOT EXISTS idx_news_status ON news(status);

-- Add check constraint to ensure valid status values
ALTER TABLE news ADD CONSTRAINT check_news_status CHECK (status IN ('draft', 'preview', 'published'));
