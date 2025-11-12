-- Migration for adding logged_at field to products table
-- This allows users to log products for specific dates (not just today)

ALTER TABLE products
ADD COLUMN logged_at DATE NOT NULL DEFAULT CURRENT_DATE;

-- Create index for faster queries by date
CREATE INDEX idx_products_logged_at ON products(logged_at);

-- Create composite index for user + date queries
CREATE INDEX idx_products_user_date ON products(user_id, logged_at);

-- Update existing records to use the date part of created_at
UPDATE products
SET logged_at = DATE(created_at);
