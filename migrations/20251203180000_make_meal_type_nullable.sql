-- Make meal_type column nullable
ALTER TABLE products
ALTER COLUMN meal_type DROP NOT NULL;

-- Remove default value
ALTER TABLE products
ALTER COLUMN meal_type DROP DEFAULT;
