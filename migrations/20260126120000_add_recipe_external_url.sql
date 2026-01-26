-- Add external_url field to recipes table for linking to external recipe sources
ALTER TABLE recipes ADD COLUMN IF NOT EXISTS external_url TEXT;

-- Add index for faster queries on recipes with external URLs
CREATE INDEX IF NOT EXISTS idx_recipes_external_url ON recipes (external_url) WHERE external_url IS NOT NULL;

-- Add comment
COMMENT ON COLUMN recipes.external_url IS 'Optional URL to an external recipe source (e.g., another website)';
