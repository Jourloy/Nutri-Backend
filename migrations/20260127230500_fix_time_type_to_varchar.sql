-- Migration to fix TIME columns that cause date corruption
-- Changes TIME to VARCHAR(5) for time-only fields to avoid lib/pq conversion issues
-- Root cause: lib/pq converts TIME to time.Time using placeholder date "0000-01-01"

-- 1. Alter supplement_schedules.intake_time from TIME to VARCHAR(5)
ALTER TABLE supplement_schedules
ALTER COLUMN intake_time TYPE VARCHAR(5) USING intake_time::TIME::VARCHAR(5);

-- 2. Alter supplement_schedules.notification_time from TIME to VARCHAR(5)
ALTER TABLE supplement_schedules
ALTER COLUMN notification_time TYPE VARCHAR(5) USING notification_time::TIME::VARCHAR(5);

-- 3. Add CHECK constraints to ensure HH:MM format (00:00 to 23:59)
ALTER TABLE supplement_schedules
ADD CONSTRAINT check_intake_time_format
CHECK (intake_time IS NULL OR intake_time ~ '^([01][0-9]|2[0-3]):[0-5][0-9]$');

ALTER TABLE supplement_schedules
ADD CONSTRAINT check_notification_time_format
CHECK (notification_time IS NULL OR notification_time ~ '^([01][0-9]|2[0-3]):[0-5][0-9]$');

-- 4. Add helpful comments
COMMENT ON COLUMN supplement_schedules.intake_time IS 'Time in HH:MM format (e.g., 08:00, 14:30, 22:50)';
COMMENT ON COLUMN supplement_schedules.notification_time IS 'Time in HH:MM format (e.g., 08:00, 14:30, 22:50)';
