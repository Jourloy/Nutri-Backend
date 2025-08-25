package user

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"

	"github.com/jourloy/nutri-backend/internal/database"
)

type Repository interface {
	CreateUser(ctx context.Context, user *UserCreate) (*User, error)
	GetUser(ctx context.Context, id string) (*User, error)
	GetUserByUsername(ctx context.Context, username string) (*User, error)
	IncreaseViewUpdates(ctx context.Context, uid string) (*User, error)
	UpdateLogin(ctx context.Context, uid string) error
	DeleteUser(ctx context.Context, id string) (*User, error)
}

type repository struct {
	db *sqlx.DB
}

func NewRepository() Repository {
	return &repository{db: database.Database}
}

// единый список колонок — не используем SELECT *
const userColumns = `
	id, username, password_hash,
	is_accept_terms, is_accept_privacy, is_18,
	telegram_chat_id, telegram_linked_at, telegram_notifications,
	token_version, view_updates, view_tutorial,
	is_admin, logined_at, created_at, updated_at, deleted_at
`

func (r *repository) CreateUser(ctx context.Context, userCreate *UserCreate) (*User, error) {
	const insertQ = `
	INSERT INTO users (username, password_hash, view_updates)
	VALUES (:username, :password_hash, :view_updates)
	ON CONFLICT (username) DO NOTHING
	RETURNING ` + userColumns + `;`

	args := map[string]any{
		"username":      userCreate.Username,
		"password_hash": userCreate.PasswordHash,
		"view_updates":  3,
	}

	// Сначала пытаемся вставить и сразу вернуть строку
	rows, err := r.db.NamedQueryContext(ctx, insertQ, args)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var u User
		if err := rows.StructScan(&u); err != nil {
			return nil, err
		}
		return &u, nil
	}

	// Если вставка не произошла (конфликт) — читаем существующего по username
	const selectQ = `SELECT ` + userColumns + ` FROM users WHERE username = $1 LIMIT 1;`
	var u User
	if err := r.db.GetContext(ctx, &u, selectQ, userCreate.Username); err != nil {
		if err == sql.ErrNoRows {
			// Теоретически не должно случиться при ON CONFLICT, но вернём nil для явности
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *repository) GetUser(ctx context.Context, id string) (*User, error) {
	const q = `SELECT ` + userColumns + ` FROM users WHERE id = $1;`

	var u User
	if err := r.db.GetContext(ctx, &u, q, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *repository) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	const q = `SELECT ` + userColumns + ` FROM users WHERE username = $1;`

	var u User
	if err := r.db.GetContext(ctx, &u, q, username); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *repository) IncreaseViewUpdates(ctx context.Context, uid string) (*User, error) {
	// Увеличиваем счётчик и возвращаем обновлённую строку
	const q = `
	UPDATE users
	SET view_updates = 3,
		updated_at   = now()
	WHERE id = $1
	RETURNING ` + userColumns + `;`

	var u User
	if err := r.db.GetContext(ctx, &u, q, uid); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *repository) UpdateLogin(ctx context.Context, uid string) error {
	const q = `
	UPDATE users
	SET logined_at = now(),
		updated_at  = now()
	WHERE id = $1
	RETURNING id;`

	var id string
	if err := r.db.GetContext(ctx, &id, q, uid); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}
	return nil
}

func (r *repository) DeleteUser(ctx context.Context, id string) (*User, error) {
	// Полное удаление. Если используешь soft-delete — замени на UPDATE ... SET deleted_at = now() RETURNING ...
	const q = `DELETE FROM users WHERE id = $1 RETURNING ` + userColumns + `;`

	var u User
	if err := r.db.GetContext(ctx, &u, q, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}
