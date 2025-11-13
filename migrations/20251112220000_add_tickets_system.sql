-- Создание таблицы тикетов
CREATE TABLE IF NOT EXISTS tickets (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL,
    subject TEXT NOT NULL,
    status TEXT DEFAULT 'open', -- 'open', 'in_progress', 'waiting_response', 'resolved', 'closed'
    priority TEXT DEFAULT 'normal', -- 'low', 'normal', 'high', 'urgent'
    category TEXT, -- 'technical', 'billing', 'feature_request', 'other'
    assigned_to UUID,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    closed_at TIMESTAMP
);

-- Создание таблицы сообщений тикета
CREATE TABLE IF NOT EXISTS ticket_messages (
    id BIGSERIAL PRIMARY KEY,
    ticket_id BIGINT NOT NULL,
    user_id UUID NOT NULL,
    message TEXT NOT NULL,
    is_admin BOOLEAN DEFAULT FALSE,
    attachments TEXT[], -- массив URL вложений
    created_at TIMESTAMP DEFAULT NOW()
);

-- Добавление foreign key constraints
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'users') THEN
        -- Foreign keys для tickets
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.table_constraints
            WHERE constraint_name = 'fk_tickets_user'
        ) THEN
            ALTER TABLE tickets
            ADD CONSTRAINT fk_tickets_user
            FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
        END IF;

        IF NOT EXISTS (
            SELECT 1 FROM information_schema.table_constraints
            WHERE constraint_name = 'fk_tickets_assigned_to'
        ) THEN
            ALTER TABLE tickets
            ADD CONSTRAINT fk_tickets_assigned_to
            FOREIGN KEY (assigned_to) REFERENCES users(id);
        END IF;

        -- Foreign key для ticket_messages -> users
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.table_constraints
            WHERE constraint_name = 'fk_ticket_messages_user'
        ) THEN
            ALTER TABLE ticket_messages
            ADD CONSTRAINT fk_ticket_messages_user
            FOREIGN KEY (user_id) REFERENCES users(id);
        END IF;
    END IF;
END $$;

-- Foreign key для ticket_messages -> tickets
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_name = 'fk_ticket_messages_ticket'
    ) THEN
        ALTER TABLE ticket_messages
        ADD CONSTRAINT fk_ticket_messages_ticket
        FOREIGN KEY (ticket_id) REFERENCES tickets(id) ON DELETE CASCADE;
    END IF;
END $$;

-- Индексы
CREATE INDEX IF NOT EXISTS idx_tickets_user_id ON tickets(user_id);
CREATE INDEX IF NOT EXISTS idx_tickets_status ON tickets(status);
CREATE INDEX IF NOT EXISTS idx_tickets_assigned_to ON tickets(assigned_to);
CREATE INDEX IF NOT EXISTS idx_tickets_created_at ON tickets(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ticket_messages_ticket_id ON ticket_messages(ticket_id);
CREATE INDEX IF NOT EXISTS idx_ticket_messages_created_at ON ticket_messages(created_at);
