-- Migration for creating the 'templates' table
CREATE TABLE
	IF NOT EXISTS templates (
		id BIGSERIAL PRIMARY KEY,
		--
		name TEXT NOT NULL, -- Название
		calories NUMERIC(6, 1) NOT NULL, -- Сколько в 100г калорий
		protein NUMERIC(6, 1) NOT NULL, -- Сколько в 100г белков
		fat NUMERIC(6, 1) NOT NULL, -- Сколько в 100г жиров
		carbs NUMERIC(6, 1) NOT NULL, -- Сколько в 100г углеводов
		--
		created_at TIMESTAMP NOT NULL DEFAULT NOW (),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW ()
	);

ALTER TABLE templates ADD CONSTRAINT templates_name_uk UNIQUE (name);