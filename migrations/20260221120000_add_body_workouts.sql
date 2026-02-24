CREATE TABLE IF NOT EXISTS body_workouts (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    logged_at DATE NOT NULL DEFAULT (CURRENT_DATE),
    duration_min INT NOT NULL,
    workout_type TEXT NOT NULL,
    intensity TEXT,
    calories_burned INT,
    note TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_body_workouts_duration CHECK (duration_min > 0 AND duration_min <= 600),
    CONSTRAINT chk_body_workouts_intensity
        CHECK (intensity IS NULL OR intensity IN ('low', 'medium', 'high')),
    CONSTRAINT chk_body_workouts_calories CHECK (calories_burned IS NULL OR calories_burned >= 0)
);

CREATE INDEX IF NOT EXISTS ix_body_workouts_user_logged_at ON body_workouts(user_id, logged_at);
CREATE INDEX IF NOT EXISTS ix_body_workouts_user_created_at ON body_workouts(user_id, created_at);
