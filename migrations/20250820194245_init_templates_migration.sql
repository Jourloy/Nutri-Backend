-- Migration for creating the 'templates' table
CREATE TABLE
	IF NOT EXISTS templates (
		id SERIAL PRIMARY KEY,
		name TEXT NOT NULL,
		calories NUMERIC(6, 1) NOT NULL,
		protein NUMERIC(6, 1) NOT NULL,
		fat NUMERIC(6, 1) NOT NULL,
		carbs NUMERIC(6, 1) NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT NOW (),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW ()
	);

ALTER TABLE templates ADD CONSTRAINT templates_name_uk UNIQUE (name);