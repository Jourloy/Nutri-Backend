-- Migration for creating the 'fit_profiles' table
CREATE TABLE
	IF NOT EXISTS fit_profiles (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
		--
		age BIGINT NOT NULL, -- Возраст
		gender TEXT NOT NULL, -- Пол
		height BIGINT NOT NULL, -- Рост
		weight BIGINT NOT NULL, -- Вес
		activity_level FLOAT NOT NULL, -- Уровень активности
		goal TEXT NOT NULL, -- Цель
		calories NUMERIC(6, 1) NOT NULL, -- Цель по калориям
		protein NUMERIC(6, 1) NOT NULL, -- Цель по белкам
		fat NUMERIC(6, 1) NOT NULL, -- Цель по жирам
		carbs NUMERIC(6, 1) NOT NULL, -- Цель по углеводам
		water_limit INT NOT NULL, -- Цель по воде
		--
		user_id UUID NOT NULL REFERENCES users (id), -- Пользователь
		--
		created_at TIMESTAMP NOT NULL DEFAULT NOW (),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW (),
		deleted_at TIMESTAMP
	);