-- Создание таблицы для хранения рассылок/уведомлений от админа
CREATE TABLE IF NOT EXISTS admin_notifications (
    id BIGSERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    message TEXT NOT NULL,
    target_audience TEXT NOT NULL, -- 'all', 'free', 'premium', 'specific'
    target_plan_id BIGINT,
    target_user_ids TEXT[], -- массив UUID пользователей для specific
    status TEXT DEFAULT 'draft', -- 'draft', 'scheduled', 'sent'
    scheduled_at TIMESTAMP,
    sent_at TIMESTAMP,
    created_by UUID,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Создание таблицы для отслеживания доставки уведомлений
CREATE TABLE IF NOT EXISTS notification_deliveries (
    id BIGSERIAL PRIMARY KEY,
    notification_id BIGINT,
    user_id UUID,
    delivered_at TIMESTAMP DEFAULT NOW(),
    read_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Добавление foreign key constraints после создания таблиц
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'plans')
       AND EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'admin_notifications' AND column_name = 'target_plan_id')
       AND NOT EXISTS (SELECT 1 FROM information_schema.table_constraints WHERE constraint_name = 'fk_admin_notifications_plan') THEN
        ALTER TABLE admin_notifications
        ADD CONSTRAINT fk_admin_notifications_plan
        FOREIGN KEY (target_plan_id) REFERENCES plans(id);
    END IF;
END $$;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'users') THEN
        IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'admin_notifications' AND column_name = 'created_by')
           AND NOT EXISTS (SELECT 1 FROM information_schema.table_constraints WHERE constraint_name = 'fk_admin_notifications_user') THEN
            ALTER TABLE admin_notifications
            ADD CONSTRAINT fk_admin_notifications_user
            FOREIGN KEY (created_by) REFERENCES users(id);
        END IF;

        IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'notification_deliveries' AND column_name = 'user_id')
           AND NOT EXISTS (SELECT 1 FROM information_schema.table_constraints WHERE constraint_name = 'fk_notification_deliveries_user') THEN
            ALTER TABLE notification_deliveries
            ADD CONSTRAINT fk_notification_deliveries_user
            FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
        END IF;
    END IF;
END $$;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'notification_deliveries' AND column_name = 'notification_id')
       AND NOT EXISTS (SELECT 1 FROM information_schema.table_constraints WHERE constraint_name = 'fk_notification_deliveries_notification') THEN
        ALTER TABLE notification_deliveries
        ADD CONSTRAINT fk_notification_deliveries_notification
        FOREIGN KEY (notification_id) REFERENCES admin_notifications(id) ON DELETE CASCADE;
    END IF;
END $$;

-- Индексы для оптимизации запросов
CREATE INDEX IF NOT EXISTS idx_admin_notifications_status ON admin_notifications(status);
CREATE INDEX IF NOT EXISTS idx_admin_notifications_created_at ON admin_notifications(created_at);
CREATE INDEX IF NOT EXISTS idx_notification_deliveries_user_id ON notification_deliveries(user_id);
CREATE INDEX IF NOT EXISTS idx_notification_deliveries_notification_id ON notification_deliveries(notification_id);

-- Добавление индекса на is_admin для быстрого поиска админов
CREATE INDEX IF NOT EXISTS idx_users_is_admin ON users(is_admin) WHERE is_admin = TRUE;

-- Добавление индекса на deleted_at для быстрого фильтрования удаленных пользователей
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at) WHERE deleted_at IS NULL;
