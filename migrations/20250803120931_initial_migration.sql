CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "citext";

CREATE TABLE
    IF NOT EXISTS users (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
        --
        username CITEXT NOT NULL, -- Никнейм
        password_hash TEXT NOT NULL, -- Хэш пароля
        --
        is_accept_terms BOOLEAN DEFAULT TRUE, -- Принял ли условия использования
        is_accept_privacy BOOLEAN DEFAULT TRUE, -- Принял ли конфиденциальность
        is_18 BOOLEAN DEFAULT TRUE, -- Есть ли 18 лет
        is_admin BOOLEAN NOT NULL DEFAULT FALSE, -- Администратор ли
        --
        token_version BIGINT DEFAULT 1, -- Версия токена
        --
        view_updates BIGINT NOT NULL DEFAULT 0, -- Какую версию обновления видел
        view_tutorial BIGINT NOT NULL DEFAULT 0, -- Какую версию туториала видел
        --
        logined_at TIMESTAMP NOT NULL DEFAULT NOW (),
        created_at TIMESTAMP NOT NULL DEFAULT NOW (),
        updated_at TIMESTAMP NOT NULL DEFAULT NOW (),
        deleted_at TIMESTAMP
    );

ALTER TABLE users ADD CONSTRAINT users_username_uk UNIQUE (username);