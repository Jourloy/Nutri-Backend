-- Migration for creating the 'fit_profiles' table
CREATE TABLE IF NOT EXISTS fit_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	age BIGINT NOT NULL,
	gender TEXT NOT NULL,
	height BIGINT NOT NULL,
	weight BIGINT NOT NULL,
	activity_level FLOAT NOT NULL,
	goal TEXT NOT NULL,
	calories BIGINT NOT NULL,
	protein BIGINT NOT NULL,
	fat BIGINT NOT NULL,
	carbs BIGINT NOT NULL,
	user_id UUID NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);