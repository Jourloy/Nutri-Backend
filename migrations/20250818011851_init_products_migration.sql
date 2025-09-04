-- Migration for creating the 'products' table
CREATE TABLE
	IF NOT EXISTS products (
		id BIGSERIAL PRIMARY KEY,
		--
		name TEXT NOT NULL, -- Название
		amount BIGINT NOT NULL, -- Количество
		unit TEXT NOT NULL, -- Единица измерения
		is_water BOOLEAN NOT NULL DEFAULT FALSE, -- Это вода?
		--
		calories NUMERIC(6, 1) NOT NULL, -- Сколько вышло калорий
		protein NUMERIC(6, 1) NOT NULL, -- Сколько вышло белков
		fat NUMERIC(6, 1) NOT NULL, -- Сколько вышло жиров
		carbs NUMERIC(6, 1) NOT NULL, --  Сколько вышло углеводов
		--
		basic_calories NUMERIC(6, 1) NOT NULL, -- Сколько в 100г калорий
		basic_protein NUMERIC(6, 1) NOT NULL, -- Сколько в 100г белков
		basic_fat NUMERIC(6, 1) NOT NULL, -- Сколько в 100г жиров
		basic_carbs NUMERIC(6, 1) NOT NULL, -- Сколько в 100г углеводов
		--
		user_id UUID NOT NULL REFERENCES users (id), -- Пользователь
		fit_id UUID NOT NULL REFERENCES fit_profiles (id), -- Профиль
		--
		created_at TIMESTAMP NOT NULL DEFAULT NOW (),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW ()
	);