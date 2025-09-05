-- Weights stored separately from measurements
CREATE TABLE IF NOT EXISTS body_weights (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    value NUMERIC(6,2) NOT NULL, -- kg
    logged_at DATE NOT NULL DEFAULT (CURRENT_DATE),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_body_weights_user_day ON body_weights(user_id, logged_at);
CREATE INDEX IF NOT EXISTS ix_body_weights_user_logged_at ON body_weights(user_id, logged_at);

-- Circumference measurements (chest/waist/hips). Any field may be NULL.
CREATE TABLE IF NOT EXISTS body_measurements (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    chest NUMERIC(6,1), -- cm
    waist NUMERIC(6,1), -- cm
    hips NUMERIC(6,1),  -- cm
    logged_at DATE NOT NULL DEFAULT (CURRENT_DATE),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_body_measurements_user_day ON body_measurements(user_id, logged_at);
CREATE INDEX IF NOT EXISTS ix_body_measurements_user_logged_at ON body_measurements(user_id, logged_at);

-- Plateau evaluation results (history)
CREATE TABLE IF NOT EXISTS body_plateau_events (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    window_start DATE NOT NULL,
    window_end DATE NOT NULL,
    goal TEXT,
    slope_weekly_pct NUMERIC(8,4) NOT NULL,
    delta_kg NUMERIC(6,3) NOT NULL,
    days_with_weight INT NOT NULL,
    calories_good_days INT NOT NULL,
    protein_good_days INT NOT NULL,
    window_days INT NOT NULL,
    is_plateau BOOLEAN NOT NULL,
    reason TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

