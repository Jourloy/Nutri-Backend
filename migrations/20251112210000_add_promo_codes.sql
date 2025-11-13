-- Создание таблицы промокодов
CREATE TABLE IF NOT EXISTS promo_codes (
    id BIGSERIAL PRIMARY KEY,
    code CITEXT UNIQUE NOT NULL,
    description TEXT,
    discount_type TEXT NOT NULL, -- 'percent' или 'fixed'
    discount_value BIGINT NOT NULL, -- процент (0-100) или сумма в копейках
    max_uses INT DEFAULT 0, -- 0 = неограниченно
    current_uses INT DEFAULT 0,
    valid_from TIMESTAMP DEFAULT NOW(),
    valid_until TIMESTAMP,
    is_active BOOLEAN DEFAULT TRUE,
    applicable_plans BIGINT[], -- массив ID тарифов, null = все тарифы
    min_amount_minor BIGINT DEFAULT 0, -- минимальная сумма заказа
    created_by UUID,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Добавление foreign key constraint для created_by
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'users')
       AND EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'promo_codes' AND column_name = 'created_by')
       AND NOT EXISTS (SELECT 1 FROM information_schema.table_constraints WHERE constraint_name = 'fk_promo_codes_user') THEN
        ALTER TABLE promo_codes
        ADD CONSTRAINT fk_promo_codes_user
        FOREIGN KEY (created_by) REFERENCES users(id);
    END IF;
END $$;

-- Добавление колонок в таблицу orders
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'orders') THEN
        ALTER TABLE orders ADD COLUMN IF NOT EXISTS promo_code_id BIGINT;
        ALTER TABLE orders ADD COLUMN IF NOT EXISTS discount_amount_minor BIGINT DEFAULT 0;
        ALTER TABLE orders ADD COLUMN IF NOT EXISTS original_amount_minor BIGINT;

        -- Добавление foreign key
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.table_constraints
            WHERE constraint_name = 'fk_orders_promo_code'
        ) THEN
            ALTER TABLE orders
            ADD CONSTRAINT fk_orders_promo_code
            FOREIGN KEY (promo_code_id) REFERENCES promo_codes(id);
        END IF;
    END IF;
END $$;

-- Добавление колонки в subscriptions
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'subscriptions') THEN
        ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS promo_code_id BIGINT;

        -- Добавление foreign key
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.table_constraints
            WHERE constraint_name = 'fk_subscriptions_promo_code'
        ) THEN
            ALTER TABLE subscriptions
            ADD CONSTRAINT fk_subscriptions_promo_code
            FOREIGN KEY (promo_code_id) REFERENCES promo_codes(id);
        END IF;
    END IF;
END $$;

-- Индексы
CREATE INDEX IF NOT EXISTS idx_promo_codes_code ON promo_codes(code);
CREATE INDEX IF NOT EXISTS idx_promo_codes_active ON promo_codes(is_active) WHERE is_active = TRUE;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'orders') THEN
        CREATE INDEX IF NOT EXISTS idx_orders_promo_code ON orders(promo_code_id);
    END IF;
END $$;
