-- Add fiber and cholesterol fields to products
ALTER TABLE products
ADD COLUMN IF NOT EXISTS fiber NUMERIC(6, 1) DEFAULT NULL,
ADD COLUMN IF NOT EXISTS cholesterol NUMERIC(6, 1) DEFAULT NULL,
ADD COLUMN IF NOT EXISTS basic_fiber NUMERIC(6, 1) DEFAULT NULL,
ADD COLUMN IF NOT EXISTS basic_cholesterol NUMERIC(6, 1) DEFAULT NULL;

-- Add fiber and cholesterol targets to fit_profiles
ALTER TABLE fit_profiles
ADD COLUMN IF NOT EXISTS fiber_target NUMERIC(6, 1) DEFAULT NULL,
ADD COLUMN IF NOT EXISTS cholesterol_limit NUMERIC(6, 1) DEFAULT NULL;
