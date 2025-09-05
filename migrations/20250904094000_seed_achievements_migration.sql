-- Seed initial categories
INSERT INTO achievement_categories (key, name, position)
VALUES
  ('streak',   'Серии',   10),
  ('calories', 'Калории', 20),
  ('protein',  'Белки',   30),
  ('special',  'Особые',  40)
ON CONFLICT (key) DO UPDATE SET
  name = EXCLUDED.name,
  position = EXCLUDED.position,
  updated_at = NOW();

-- Seed initial achievements (without prerequisites first)
INSERT INTO achievements (key, category_id, name, description, icon, color, points, is_secret, enabled, criteria)
VALUES
  ('first_product', (SELECT id FROM achievement_categories WHERE key='streak'),
    'Первый шаг', 'Добавьте первую запись о питании', 'star', 'from-yellow-400 to-yellow-500', 10, FALSE, TRUE,
    '{"metric":"total_products_count","threshold":1}'::jsonb),

  ('week_streak', (SELECT id FROM achievement_categories WHERE key='streak'),
    'Неделя силы', 'Ведите дневник питания 7 дней подряд', 'flame', 'from-orange-400 to-red-500', 20, FALSE, TRUE,
    '{"metric":"daily_streak_products","threshold":7,"consecutive":true}'::jsonb),

  ('month_streak', (SELECT id FROM achievement_categories WHERE key='streak'),
    'Месячный марафон', 'Ведите дневник питания 30 дней подряд', 'trophy', 'from-purple-400 to-purple-600', 40, FALSE, TRUE,
    '{"metric":"daily_streak_products","threshold":30,"consecutive":true}'::jsonb),

  ('cal_100', (SELECT id FROM achievement_categories WHERE key='calories'),
    'Первая сотня', 'Отследите 100 калорий', 'target', 'from-green-400 to-emerald-500', 10, FALSE, TRUE,
    '{"metric":"total_calories_sum","threshold":100}'::jsonb),

  ('cal_1000', (SELECT id FROM achievement_categories WHERE key='calories'),
    'Первая тысяча', 'Отследите 1000 калорий', 'zap', 'from-blue-400 to-cyan-500', 20, FALSE, TRUE,
    '{"metric":"total_calories_sum","threshold":1000}'::jsonb),

  ('prot_100', (SELECT id FROM achievement_categories WHERE key='protein'),
    'Любитель белка', 'Отследите 100г белка', 'heart', 'from-red-400 to-pink-500', 10, FALSE, TRUE,
    '{"metric":"total_protein_sum","threshold":100}'::jsonb),

  ('prot_1000', (SELECT id FROM achievement_categories WHERE key='protein'),
    'Мастер белка', 'Отследите 1000г белка', 'medal', 'from-emerald-400 to-green-600', 30, FALSE, TRUE,
    '{"metric":"total_protein_sum","threshold":1000}'::jsonb),

  ('legend', (SELECT id FROM achievement_categories WHERE key='special'),
    'Легенда питания', 'Получите все основные награды', 'crown', 'from-amber-400 to-yellow-600', 100, TRUE, TRUE,
    '{"metric":"total_products_count","threshold":1}'::jsonb)
ON CONFLICT (key) DO UPDATE SET
  category_id = EXCLUDED.category_id,
  name = EXCLUDED.name,
  description = EXCLUDED.description,
  icon = EXCLUDED.icon,
  color = EXCLUDED.color,
  points = EXCLUDED.points,
  is_secret = EXCLUDED.is_secret,
  enabled = EXCLUDED.enabled,
  criteria = EXCLUDED.criteria,
  updated_at = NOW();

-- Set prerequisites now that all base rows exist
UPDATE achievements a SET prerequisite_id = b.id
FROM achievements b
WHERE a.key='cal_1000' AND b.key='cal_100';

UPDATE achievements a SET prerequisite_id = b.id
FROM achievements b
WHERE a.key='prot_1000' AND b.key='prot_100';

UPDATE achievements a SET prerequisite_id = b.id
FROM achievements b
WHERE a.key='month_streak' AND b.key='week_streak';

UPDATE achievements a SET prerequisite_id = b.id
FROM achievements b
WHERE a.key='legend' AND b.key='month_streak';

