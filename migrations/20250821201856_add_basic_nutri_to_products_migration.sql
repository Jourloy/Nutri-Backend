ALTER TABLE products
ADD COLUMN basic_calories NUMERIC(6, 1),
ADD COLUMN basic_protein NUMERIC(6, 1),
ADD COLUMN basic_fat NUMERIC(6, 1),
ADD COLUMN basic_carbs NUMERIC(6, 1);

UPDATE products SET basic_calories = 0;
UPDATE products SET basic_protein = 0;
UPDATE products SET basic_fat = 0;
UPDATE products SET basic_carbs = 0;

ALTER TABLE products ALTER COLUMN basic_calories SET NOT NULL;
ALTER TABLE products ALTER COLUMN basic_protein SET NOT NULL;
ALTER TABLE products ALTER COLUMN basic_fat SET NOT NULL;
ALTER TABLE products ALTER COLUMN basic_carbs SET NOT NULL;