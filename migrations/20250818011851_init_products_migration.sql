-- Migration for creating the 'products' table
CREATE TABLE IF NOT EXISTS products (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	name TEXT NOT NULL,
	amount BIGINT NOT NULL,
	unit TEXT NOT NULL,
	calories BIGINT NOT NULL,
	protein BIGINT NOT NULL,
	fat BIGINT NOT NULL,
	carbs BIGINT NOT NULL,
	user_id UUID NOT NULL,
	fit_id UUID NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);