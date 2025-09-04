CREATE TABLE
	IF NOT EXISTS telegram_profiles (
		id BIGSERIAL PRIMARY KEY,
		--
		token TEXT NOT NULL DEFAULT gen_random_uuid (),
		--
		telegram_id TEXT, -- ID
		telegram_username TEXT, -- Никнейм
		telegram_avatar TEXT, -- Ссылка на аватарку
		--
		notify_daily BOOLEAN NOT NULL DEFAULT true, -- Отправлять ли дневные уведомления
		notify_story BOOLEAN NOT NULL DEFAULT true, -- Отправлять ли уведомления про истории
		--
		user_id UUID REFERENCES users (id) NOT NULl,
		--
		connected_at TIMESTAMP,
		created_at TIMESTAMP NOT NULL DEFAULT now (),
		updated_at TIMESTAMP NOT NULL DEFAULT now ()
	);