CREATE EXTENSION IF NOT EXISTS citext;

-- меняем тип на citext (автоматически case-insensitive сравнение/уникальность)
ALTER TABLE users
ALTER COLUMN username TYPE citext;

ALTER TABLE users
ALTER COLUMN username
SET
	NOT NULL;

ALTER TABLE users ADD CONSTRAINT users_username_uk UNIQUE (username);