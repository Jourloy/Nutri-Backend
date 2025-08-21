-- Migration for creating the 'fit_profiles' table
CREATE TABLE IF NOT EXISTS fit_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	age BIGINT NOT NULL,
	gender TEXT NOT NULL,
	height BIGINT NOT NULL,
	weight BIGINT NOT NULL,
	activity_level FLOAT NOT NULL,
	goal TEXT NOT NULL,
	calories NUMERIC(6,1) NOT NULL,
	protein NUMERIC(6,1) NOT NULL,
	fat NUMERIC(6,1) NOT NULL,
	carbs NUMERIC(6,1) NOT NULL,
	user_id UUID NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);