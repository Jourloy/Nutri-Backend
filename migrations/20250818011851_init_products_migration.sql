-- Migration for creating the 'products' table
CREATE TABLE IF NOT EXISTS products (
    id SERIAL PRIMARY KEY,
	name TEXT NOT NULL,
	amount BIGINT NOT NULL,
	unit TEXT NOT NULL,
	calories NUMERIC(6,1) NOT NULL,
	protein NUMERIC(6,1) NOT NULL,
	fat NUMERIC(6,1) NOT NULL,
	carbs NUMERIC(6,1) NOT NULL,
	user_id UUID NOT NULL,
	fit_id UUID NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);