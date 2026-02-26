-- Add preview status for blog articles.

DO $$
DECLARE
	existing_constraint_name TEXT;
BEGIN
	SELECT conname
	INTO existing_constraint_name
	FROM pg_constraint
	WHERE conrelid = 'blog_articles'::regclass
		AND contype = 'c'
		AND pg_get_constraintdef(oid) ILIKE '%status%'
		AND pg_get_constraintdef(oid) ILIKE '%draft%'
		AND pg_get_constraintdef(oid) ILIKE '%authorized%'
		AND pg_get_constraintdef(oid) ILIKE '%paid%'
		AND pg_get_constraintdef(oid) ILIKE '%public%'
	LIMIT 1;

	IF existing_constraint_name IS NOT NULL THEN
		EXECUTE format('ALTER TABLE blog_articles DROP CONSTRAINT %I', existing_constraint_name);
	END IF;
END $$;

ALTER TABLE blog_articles
	ADD CONSTRAINT blog_articles_status_check
	CHECK (status IN ('draft', 'preview', 'authorized', 'paid', 'public'));
