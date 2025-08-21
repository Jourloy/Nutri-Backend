CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Migration for creating the 'users' table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    is_accept_terms BOOLEAN DEFAULT TRUE,
    is_accept_privacy BOOLEAN DEFAULT TRUE,
    is_18 BOOLEAN DEFAULT TRUE,
    telegram_chat_id TEXT,
    telegram_linked_at TIMESTAMP,
    telegram_notifications BOOLEAN DEFAULT FALSE,
    token_version BIGINT DEFAULT 1,
    view_updates BIGINT DEFAULT 0,
    view_tutorial BIGINT DEFAULT 0,
    logined_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);