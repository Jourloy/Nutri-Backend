-- Daily activity: steps and sleep minutes
CREATE TABLE IF NOT EXISTS body_activity (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    steps INT,
    sleep_min INT,
    logged_at DATE NOT NULL DEFAULT (CURRENT_DATE),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_body_activity_user_day ON body_activity(user_id, logged_at);
CREATE INDEX IF NOT EXISTS ix_body_activity_user_logged_at ON body_activity(user_id, logged_at);

-- Steps target in fit profile (default 8000)
ALTER TABLE fit_profiles
    ADD COLUMN IF NOT EXISTS steps_target INT NOT NULL DEFAULT 8000;

