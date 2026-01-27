-- 1. supplements_templates table
-- Predefined supplement templates (vitamins, minerals, sports nutrition)
CREATE TABLE IF NOT EXISTS supplements_templates (
    id BIGSERIAL PRIMARY KEY,
    slug VARCHAR(100) UNIQUE NOT NULL,
    name_ru VARCHAR(255) NOT NULL,
    name_en VARCHAR(255) NOT NULL,
    category VARCHAR(50) NOT NULL, -- 'vitamin', 'mineral', 'sports', 'other'
    icon VARCHAR(100),
    description_ru TEXT,
    description_en TEXT,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_supplements_templates_category ON supplements_templates(category);
CREATE INDEX idx_supplements_templates_sort ON supplements_templates(sort_order);

-- 2. supplements table
-- User's supplements/medications
CREATE TABLE IF NOT EXISTS supplements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- Template OR custom name
    template_id BIGINT REFERENCES supplements_templates(id) ON DELETE SET NULL,
    custom_name VARCHAR(255), -- NULL if using template

    -- Start date
    start_date DATE NOT NULL,
    end_date DATE, -- NULL = no end date

    -- Status
    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    -- Metadata
    notes TEXT, -- User notes

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP,

    CONSTRAINT check_name_source CHECK (
        (template_id IS NOT NULL AND custom_name IS NULL) OR
        (template_id IS NULL AND custom_name IS NOT NULL)
    )
);

CREATE INDEX idx_supplements_user ON supplements(user_id) WHERE deleted_at IS NULL AND is_active = TRUE;
CREATE INDEX idx_supplements_template ON supplements(template_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_supplements_start_date ON supplements(start_date) WHERE deleted_at IS NULL;
CREATE INDEX idx_supplements_user_active ON supplements(user_id, is_active) WHERE deleted_at IS NULL;

-- 3. supplement_schedules table
-- Defines when to take supplements
CREATE TABLE IF NOT EXISTS supplement_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    supplement_id UUID NOT NULL REFERENCES supplements(id) ON DELETE CASCADE,

    -- Frequency type
    frequency_type VARCHAR(50) NOT NULL, -- 'times_per_day', 'once_per_day', 'every_n_days', 'once_per_week', 'once_per_month'

    -- For 'times_per_day' and 'once_per_day': specific time
    intake_time TIME, -- HH:MM (e.g., '08:00', '14:00', '20:00')

    -- Days of week (for 'times_per_day', 'once_per_day', 'once_per_week')
    -- Array: [1,2,3,4,5] = Mon-Fri, [0,6] = Sun,Sat (0=Sunday, 6=Saturday)
    days_of_week INTEGER[], -- PostgreSQL array

    -- For 'every_n_days'
    interval_days INTEGER, -- Take every N days

    -- For 'once_per_month'
    day_of_month INTEGER, -- 1-31

    -- Notification settings
    enable_notification BOOLEAN NOT NULL DEFAULT TRUE,
    notification_time TIME, -- For non-exact time frequencies (morning reminder)

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT check_frequency_data CHECK (
        (frequency_type = 'times_per_day' AND intake_time IS NOT NULL) OR
        (frequency_type = 'once_per_day') OR
        (frequency_type = 'every_n_days' AND interval_days IS NOT NULL AND interval_days > 0) OR
        (frequency_type = 'once_per_week' AND days_of_week IS NOT NULL AND array_length(days_of_week, 1) = 1) OR
        (frequency_type = 'once_per_month' AND day_of_month BETWEEN 1 AND 31)
    )
);

CREATE INDEX idx_supplement_schedules_supplement ON supplement_schedules(supplement_id);
CREATE INDEX idx_supplement_schedules_frequency ON supplement_schedules(frequency_type);
CREATE INDEX idx_supplement_schedules_notification ON supplement_schedules(enable_notification) WHERE enable_notification = TRUE;

-- 4. supplement_intakes table
-- Actual intake records
CREATE TABLE IF NOT EXISTS supplement_intakes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    supplement_id UUID NOT NULL REFERENCES supplements(id) ON DELETE CASCADE,
    schedule_id UUID REFERENCES supplement_schedules(id) ON DELETE SET NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- Scheduled vs actual
    scheduled_at TIMESTAMP NOT NULL, -- When it was supposed to be taken
    taken_at TIMESTAMP NOT NULL, -- When it was actually taken (user timezone)

    -- Was it taken on time?
    is_on_time BOOLEAN NOT NULL DEFAULT TRUE,
    is_missed BOOLEAN NOT NULL DEFAULT FALSE, -- Marked as missed (not taken)

    -- Source
    source VARCHAR(50) NOT NULL DEFAULT 'manual', -- 'manual', 'telegram', 'dashboard'

    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_supplement_intakes_supplement ON supplement_intakes(supplement_id);
CREATE INDEX idx_supplement_intakes_user_date ON supplement_intakes(user_id, DATE(scheduled_at));
CREATE INDEX idx_supplement_intakes_scheduled ON supplement_intakes(scheduled_at);
CREATE INDEX idx_supplement_intakes_missed ON supplement_intakes(is_missed) WHERE is_missed = TRUE;
CREATE INDEX idx_supplement_intakes_user_supplement ON supplement_intakes(user_id, supplement_id);

-- 5. supplement_notification_log table
-- Track sent notifications to prevent duplicates
CREATE TABLE IF NOT EXISTS supplement_notification_log (
    id BIGSERIAL PRIMARY KEY,
    supplement_id UUID NOT NULL REFERENCES supplements(id) ON DELETE CASCADE,
    schedule_id UUID NOT NULL REFERENCES supplement_schedules(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    telegram_id VARCHAR(50) NOT NULL,

    scheduled_for TIMESTAMP NOT NULL, -- The exact time this notification was for
    sent_at TIMESTAMP NOT NULL DEFAULT NOW(),
    notification_type VARCHAR(50) NOT NULL, -- 'exact_time', 'morning_reminder', 'missed_reminder'

    CONSTRAINT unique_notification UNIQUE (schedule_id, scheduled_for, notification_type)
);

CREATE INDEX idx_supplement_notification_log_user ON supplement_notification_log(user_id);
CREATE INDEX idx_supplement_notification_log_sent ON supplement_notification_log(sent_at);
CREATE INDEX idx_supplement_notification_log_schedule ON supplement_notification_log(schedule_id, scheduled_for);

-- Seed supplement templates
INSERT INTO supplements_templates (slug, name_ru, name_en, category, icon, sort_order) VALUES
-- Vitamins
('vitamin-d3', 'Витамин D3', 'Vitamin D3', 'vitamin', '☀️', 1),
('vitamin-c', 'Витамин C', 'Vitamin C', 'vitamin', '🍊', 2),
('vitamin-b12', 'Витамин B12', 'Vitamin B12', 'vitamin', '💊', 3),
('omega-3', 'Омега-3', 'Omega-3', 'vitamin', '🐟', 4),

-- Minerals
('magnesium', 'Магний', 'Magnesium', 'mineral', '⚡', 5),
('calcium', 'Кальций', 'Calcium', 'mineral', '🦴', 6),
('iron', 'Железо', 'Iron', 'mineral', '🔴', 7),
('zinc', 'Цинк', 'Zinc', 'mineral', '⚪', 8),

-- Sports nutrition
('protein', 'Протеин', 'Protein', 'sports', '💪', 9),
('creatine', 'Креатин', 'Creatine', 'sports', '🏋️', 10),
('bcaa', 'BCAA', 'BCAA', 'sports', '🥤', 11)
ON CONFLICT (slug) DO NOTHING;
