-- plans
CREATE TABLE
	IF NOT EXISTS plans (
		id BIGSERIAL PRIMARY KEY,
		code CITEXT UNIQUE NOT NULL, -- "FREE", "PLUS", "PRO", "ELITE", "COACH"
		name TEXT NOT NULL, -- человекочитаемое
		plan_type TEXT NOT NULL DEFAULT 'consumer', -- 'consumer' | 'coach'
		version INT NOT NULL DEFAULT 1, -- для «грандфазеринга»
		currency TEXT NOT NULL DEFAULT 'RUB',
		amount_minor BIGINT NOT NULL, -- цена за период в рублях
		billing_period TEXT NOT NULL DEFAULT 'month', -- 'month' | 'year'
		trial_days INT NOT NULL DEFAULT 0,
		client_limit INT NOT NULL DEFAULT 0, -- лимит клиентов для 'coach' планов
		is_active BOOLEAN NOT NULL DEFAULT TRUE,
		created_at TIMESTAMP NOT NULL DEFAULT now (),
		updated_at TIMESTAMP NOT NULL DEFAULT now (),
		external_product_id TEXT, -- для YooKassa/CloudPayments/Stripe и т.п.
		external_price_id TEXT
	);

-- features: каталог
CREATE TABLE
	IF NOT EXISTS features (
		key TEXT PRIMARY KEY, -- 'ai_insights', 'barcode_scan', ...
		name TEXT NOT NULL,
		description TEXT NOT NULL,
		unit TEXT NOT NULL DEFAULT 'flag' -- 'flag' | 'count' | 'per_day' | ...
	);

-- plan_features: значения фич/лимитов для плана
CREATE TABLE
	IF NOT EXISTS plan_features (
		plan_id BIGINT REFERENCES plans (id) ON DELETE CASCADE,
		feature_key TEXT REFERENCES features (key) ON DELETE CASCADE,
		value JSONB NOT NULL, -- { "enabled": true } или { "limit": 100 }
		PRIMARY KEY (plan_id, feature_key)
	);

-- subscriptions: подписки пользователей
CREATE TABLE
	IF NOT EXISTS subscriptions (
		id BIGSERIAL PRIMARY KEY,
		user_id UUID NOT NULL, -- ваш users.id (UUID/строка)
		plan_id BIGINT NOT NULL REFERENCES plans (id),
		status TEXT NOT NULL, -- 'trialing'|'active'|'past_due'|'canceled'|'incomplete'
		period_start TIMESTAMP NOT NULL,
		period_end TIMESTAMP NOT NULL,
		cancel_at TIMESTAMP, -- запланированная отмена
		canceled_at TIMESTAMP,
		trial_end TIMESTAMP,
		amount_minor BIGINT NOT NULL, -- зафиксированная цена (на момент покупки)
		currency TEXT NOT NULL DEFAULT 'RUB',
		billing_period TEXT NOT NULL DEFAULT 'month',
		external_subscription_id TEXT,
		external_customer_id TEXT,
		created_at TIMESTAMP NOT NULL DEFAULT now (),
		updated_at TIMESTAMP NOT NULL DEFAULT now ()
	);

-- одна активная/пробная подписка на пользователя
CREATE UNIQUE INDEX IF NOT EXISTS uniq_active_sub_per_user ON subscriptions (user_id)
WHERE
	status IN ('trialing', 'active', 'past_due');

-- coach_clients: связь тренера и клиента (по согласию)
CREATE TABLE
	IF NOT EXISTS coach_clients (
		coach_user_id TEXT NOT NULL,
		client_user_id TEXT NOT NULL,
		state TEXT NOT NULL DEFAULT 'active', -- 'active'|'revoked'
		created_at TIMESTAMP NOT NULL DEFAULT now (),
		accepted_at TIMESTAMP, -- момент подтверждения клиентом
		revoked_at TIMESTAMP,
		PRIMARY KEY (coach_user_id, client_user_id)
	);

-- features: seed
INSERT INTO
	features (key, name, description, unit)
VALUES
	(
		'data',
		'Данные',
		'Исторические данные за период',
		'count'
	),
	(
		'recipes',
		'Рецепты',
		'Сохранение и расчёт блюд',
		'count'
	),
	(
		'export',
		'Экспорт',
		'Экспорт данных в CSV',
		'flag'
	),
	(
		'telegram_bot',
		'Telegram бот',
		'Доступ к боту в Telegram',
		'flag'
	),
	(
		'body_measurements',
		'Измерения тела',
		'Сохранение и расчёт измерений тела',
		'flag'
	),
	(
		'dynamic_goals',
		'Гибкие цели',
		'Установка целей на основе данных',
		'flag'
	),
	(
		'templates_limit',
		'База продуктов',
		'Доступ к базе продуктов',
		'flag'
	),
	(
		'plateau_detector',
		'Детектор плато',
		'Определяет застой по прогрессу',
		'flag'
	),
	(
		'daily_notes',
		'Дневник самочувствия',
		'Заметки за день',
		'flag'
	),
	(
		'recipes_library',
		'Библиотека рецептов',
		'Доступ к нашей библиотеке рецептов',
		'flag'
	),
	(
		'ai_insights',
		'Нейросеть',
		'Анализ пищевого поведения и подсказки ИИ',
		'flag'
	),
	(
		'coach_dashboard',
		'Кабинет тренера',
		'Просмотр клиентов',
		'flag'
	) ON CONFLICT (key) DO NOTHING;

-- 4 обычных + 1 тренерский
INSERT INTO
	plans (
		code,
		name,
		plan_type,
		version,
		amount_minor,
		billing_period,
		trial_days,
		client_limit
	)
VALUES
	('START', 'Старт', 'consumer', 1, 0, 'month', 0, 0),
	(
		'BALANCE',
		'Баланс',
		'consumer',
		1,
		349,
		'month',
		0,
		0
	),
	(
		'BALANCE_YEAR',
		'Баланс (год)',
		'consumer',
		1,
		3490,
		'year',
		0,
		0
	),
	(
		'RESULT',
		'Результат',
		'consumer',
		1,
		699,
		'month',
		0,
		0
	),
	(
		'RESULT_YEAR',
		'Результат (год)',
		'consumer',
		1,
		6990,
		'year',
		0,
		0
	),
	(
		'NEURO',
		'Нейроанализ',
		'consumer',
		1,
		1990,
		'month',
		0,
		0
	),
	(
		'NEURO_YEAR',
		'Нейроанализ (год)',
		'consumer',
		1,
		19900,
		'year',
		0,
		0
	),
	(
		'COACH',
		'Тренер',
		'coach',
		1,
		1990,
		'month',
		0,
		30
	) ON CONFLICT (code) DO NOTHING;

-- START
INSERT INTO
	plan_features (plan_id, feature_key, value)
SELECT
	id,
	'data',
	'{"limit": 7}'
FROM
	plans
WHERE
	code = 'START';

-- BALANCE
INSERT INTO
	plan_features (plan_id, feature_key, value)
SELECT
	id,
	'data',
	'{"limit": 31}'
FROM
	plans
WHERE
	code = 'BALANCE';

INSERT INTO
	plan_features (plan_id, feature_key, value)
SELECT
	id,
	'recipes',
	'{"limit": 10}'
FROM
	plans
WHERE
	code = 'BALANCE';

INSERT INTO
	plan_features (plan_id, feature_key, value)
SELECT
	id,
	'export',
	'{"enabled": true}'
FROM
	plans
WHERE
	code = 'BALANCE';

INSERT INTO
	plan_features (plan_id, feature_key, value)
SELECT
	id,
	'telegram_bot',
	'{"enabled": true}'
FROM
	plans
WHERE
	code = 'BALANCE';

INSERT INTO
	plan_features (plan_id, feature_key, value)
SELECT
	id,
	'templates_limit',
	'{"enabled": true}'
FROM
	plans
WHERE
	code = 'BALANCE';

-- RESULT
INSERT INTO
	plan_features (plan_id, feature_key, value)
SELECT
	id,
	'data',
	'{"limit": -1}'
FROM
	plans
WHERE
	code = 'RESULT';

INSERT INTO
	plan_features (plan_id, feature_key, value)
SELECT
	id,
	'recipes',
	'{"limit": -1}'
FROM
	plans
WHERE
	code = 'RESULT';

INSERT INTO
	plan_features (plan_id, feature_key, value)
SELECT
	id,
	'export',
	'{"enabled": true}'
FROM
	plans
WHERE
	code = 'RESULT';

INSERT INTO
	plan_features (plan_id, feature_key, value)
SELECT
	id,
	'telegram_bot',
	'{"enabled": true}'
FROM
	plans
WHERE
	code = 'RESULT';

INSERT INTO
	plan_features (plan_id, feature_key, value)
SELECT
	id,
	'body_measurements',
	'{"enabled": true}'
FROM
	plans
WHERE
	code = 'RESULT';

INSERT INTO
	plan_features (plan_id, feature_key, value)
SELECT
	id,
	'dynamic_goals',
	'{"enabled": true}'
FROM
	plans
WHERE
	code = 'RESULT';

INSERT INTO
	plan_features (plan_id, feature_key, value)
SELECT
	id,
	'plateau_detector',
	'{"enabled": true}'
FROM
	plans
WHERE
	code = 'RESULT';

INSERT INTO
	plan_features (plan_id, feature_key, value)
SELECT
	id,
	'templates_limit',
	'{"enabled": true}'
FROM
	plans
WHERE
	code = 'RESULT';

INSERT INTO
	plan_features (plan_id, feature_key, value)
SELECT
	id,
	'daily_notes',
	'{"enabled": true}'
FROM
	plans
WHERE
	code = 'RESULT';

INSERT INTO
	plan_features (plan_id, feature_key, value)
SELECT
	id,
	'recipes_library',
	'{"enabled": true}'
FROM
	plans
WHERE
	code = 'RESULT';