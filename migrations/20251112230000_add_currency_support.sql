-- Добавление поддержки мультивалютности

-- Таблица курсов валют (для конвертации)
CREATE TABLE IF NOT EXISTS currency_rates (
    id BIGSERIAL PRIMARY KEY,
    from_currency TEXT NOT NULL,
    to_currency TEXT NOT NULL,
    rate NUMERIC(10, 4) NOT NULL,
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(from_currency, to_currency)
);

-- Вставка базовых курсов (будут обновляться автоматически)
INSERT INTO currency_rates (from_currency, to_currency, rate) VALUES
    ('RUB', 'USD', 0.0105),
    ('USD', 'RUB', 95.0),
    ('RUB', 'RUB', 1.0),
    ('USD', 'USD', 1.0)
ON CONFLICT (from_currency, to_currency) DO NOTHING;

-- Добавление тарифов в долларах (копии существующих, но в USD)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'plans') THEN
        -- Предполагается, что уже есть тарифы в RUB
        INSERT INTO plans (code, name, plan_type, version, currency, amount_minor, billing_period, trial_days, is_active, client_limit)
        SELECT
            code || '_USD' as code,
            name,
            plan_type,
            version,
            'USD' as currency,
            ROUND(amount_minor * 0.0105) as amount_minor, -- конвертация в центы USD
            billing_period,
            trial_days,
            is_active,
            client_limit
        FROM plans
        WHERE currency = 'RUB'
        ON CONFLICT (code) DO NOTHING;
    END IF;
END $$;

-- Индексы для быстрого поиска по валюте
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'plans') THEN
        CREATE INDEX IF NOT EXISTS idx_plans_currency ON plans(currency);
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'orders') THEN
        CREATE INDEX IF NOT EXISTS idx_orders_currency ON orders(currency);
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'subscriptions') THEN
        CREATE INDEX IF NOT EXISTS idx_subscriptions_currency ON subscriptions(currency);
    END IF;
END $$;
