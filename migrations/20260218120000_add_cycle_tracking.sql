-- Cycle tracking: menstrual cycles, daily logs, and daily cycle events
CREATE TABLE IF NOT EXISTS body_cycles (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    start_date DATE NOT NULL,
    end_date DATE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_body_cycles_end_after_start CHECK (end_date IS NULL OR end_date >= start_date)
);

CREATE INDEX IF NOT EXISTS ix_body_cycles_user_start_date ON body_cycles(user_id, start_date);
CREATE INDEX IF NOT EXISTS ix_body_cycles_user_end_date ON body_cycles(user_id, end_date);
CREATE UNIQUE INDEX IF NOT EXISTS ux_body_cycles_user_open_cycle ON body_cycles(user_id) WHERE end_date IS NULL;

CREATE TABLE IF NOT EXISTS body_cycle_daily_logs (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    logged_at DATE NOT NULL DEFAULT (CURRENT_DATE),
    flow_intensity TEXT,
    note TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_body_cycle_daily_logs_flow_intensity
        CHECK (flow_intensity IS NULL OR flow_intensity IN ('spotting', 'light', 'medium', 'heavy'))
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_body_cycle_daily_logs_user_day ON body_cycle_daily_logs(user_id, logged_at);
CREATE INDEX IF NOT EXISTS ix_body_cycle_daily_logs_user_logged_at ON body_cycle_daily_logs(user_id, logged_at);

CREATE TABLE IF NOT EXISTS body_cycle_daily_events (
    id BIGSERIAL PRIMARY KEY,
    day_log_id BIGINT NOT NULL REFERENCES body_cycle_daily_logs(id) ON DELETE CASCADE,
    event_category TEXT NOT NULL,
    event_code TEXT NOT NULL,
    quantity INT NOT NULL DEFAULT 1,
    intensity TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_body_cycle_daily_events_quantity CHECK (quantity > 0),
    CONSTRAINT chk_body_cycle_daily_events_intensity
        CHECK (intensity IS NULL OR intensity IN ('low', 'medium', 'high'))
);

CREATE INDEX IF NOT EXISTS ix_body_cycle_daily_events_day_log_id ON body_cycle_daily_events(day_log_id);
CREATE INDEX IF NOT EXISTS ix_body_cycle_daily_events_category_code ON body_cycle_daily_events(event_category, event_code);
