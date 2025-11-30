-- Add English translation columns to achievements table
ALTER TABLE achievements ADD COLUMN IF NOT EXISTS name_en TEXT;
ALTER TABLE achievements ADD COLUMN IF NOT EXISTS description_en TEXT;

-- Add English translation column to achievement_categories table
ALTER TABLE achievement_categories ADD COLUMN IF NOT EXISTS name_en TEXT;

-- Update achievements with English translations
UPDATE achievements SET
    name_en = 'First Step',
    description_en = 'Add your first nutrition entry'
WHERE key = 'first_product';

UPDATE achievements SET
    name_en = 'Week of Strength',
    description_en = 'Log meals for 7 days in a row'
WHERE key = 'week_streak';

UPDATE achievements SET
    name_en = 'Monthly Marathon',
    description_en = 'Log meals for 30 days in a row'
WHERE key = 'month_streak';

UPDATE achievements SET
    name_en = 'First Hundred',
    description_en = 'Track 100 calories'
WHERE key = 'cal_100';

UPDATE achievements SET
    name_en = 'First Thousand',
    description_en = 'Track 1000 calories'
WHERE key = 'cal_1000';

UPDATE achievements SET
    name_en = 'Protein Lover',
    description_en = 'Track 100g of protein'
WHERE key = 'prot_100';

UPDATE achievements SET
    name_en = 'Protein Master',
    description_en = 'Track 1000g of protein'
WHERE key = 'prot_1000';

UPDATE achievements SET
    name_en = 'Nutrition Legend',
    description_en = 'Unlock all main achievements'
WHERE key = 'legend';

-- Update achievement categories with English translations
UPDATE achievement_categories SET name_en = 'Streaks' WHERE key = 'streak';
UPDATE achievement_categories SET name_en = 'Calories' WHERE key = 'calories';
UPDATE achievement_categories SET name_en = 'Protein' WHERE key = 'protein';
UPDATE achievement_categories SET name_en = 'Special' WHERE key = 'special';
