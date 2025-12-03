-- Migration for adding meal_type field to products table
ALTER TABLE products
ADD COLUMN meal_type VARCHAR(20) DEFAULT 'snack' NOT NULL
CHECK (meal_type IN ('breakfast', 'lunch', 'dinner', 'snack'));

-- Create indexes for performance
CREATE INDEX idx_products_meal_type ON products(meal_type);
CREATE INDEX idx_products_user_date_meal ON products(user_id, logged_at, meal_type);
