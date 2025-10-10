ALTER TABLE feedbacks
    ADD COLUMN IF NOT EXISTS viewed BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_feedbacks_viewed_created_at
    ON feedbacks (viewed, created_at DESC);
