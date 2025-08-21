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
	DeleteUser(ctx context.Context, id string) (*User, error)
}

type repository struct {
	db *sqlx.DB
}

func NewRepository() Repository {
	return &repository{db: database.Database}
}

func (r *repository) CreateUser(ctx context.Context, userCreate *UserCreate) (*User, error) {
	user := User{
		Username: userCreate.Username,
	}

	// Проверяем, существует ли пользователь
	query := "SELECT * FROM users WHERE username = $1"
	err := r.db.QueryRowContext(ctx, query, user.Username).Scan(&user.Id, &user.Username, &user.PasswordHash, &user.IsAcceptTerms, &user.IsAcceptPrivacy, &user.Is18, &user.TelegramChatId, &user.TelegramLinkedAt, &user.TelegramNotifications, &user.TokenVersion, &user.ViewUpdates, &user.ViewTutorial, &user.LoginedAt, &user.CreatedAt, &user.UpdatedAt, &user.DeletedAt)
	if err != nil && err != sql.ErrNoRows {
		logger.Error(err)
		return nil, err
	}

	if user.Id != "" {
		return &user, nil
	}

	user = User{
		Username:     userCreate.Username,
		PasswordHash: userCreate.PasswordHash,
		ViewUpdates:  1,
	}

	query = `INSERT INTO users (
	username, 
	password_hash,
	view_updates
	) VALUES (
	$1, $2, $3
	) RETURNING *`

	err = r.db.QueryRowContext(ctx, query, user.Username, user.PasswordHash, user.ViewUpdates).Scan(&user.Id, &user.Username, &user.PasswordHash, &user.IsAcceptTerms, &user.IsAcceptPrivacy, &user.Is18, &user.TelegramChatId, &user.TelegramLinkedAt, &user.TelegramNotifications, &user.TokenVersion, &user.ViewUpdates, &user.ViewTutorial, &user.LoginedAt, &user.CreatedAt, &user.UpdatedAt, &user.DeletedAt)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *repository) GetUser(ctx context.Context, id string) (*User, error) {
	query := "SELECT * FROM users WHERE id = $1"
	row := r.db.QueryRowContext(ctx, query, id)

	var user User
	err := row.Scan(&user.Id, &user.Username, &user.PasswordHash, &user.IsAcceptTerms, &user.IsAcceptPrivacy, &user.Is18, &user.TelegramChatId, &user.TelegramLinkedAt, &user.TelegramNotifications, &user.TokenVersion, &user.ViewUpdates, &user.ViewTutorial, &user.LoginedAt, &user.CreatedAt, &user.UpdatedAt, &user.DeletedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *repository) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	query := "SELECT * FROM users WHERE username = $1"
	row := r.db.QueryRowContext(ctx, query, username)

	var user User
	err := row.Scan(&user.Id, &user.Username, &user.PasswordHash, &user.IsAcceptTerms, &user.IsAcceptPrivacy, &user.Is18, &user.TelegramChatId, &user.TelegramLinkedAt, &user.TelegramNotifications, &user.TokenVersion, &user.ViewUpdates, &user.ViewTutorial, &user.LoginedAt, &user.CreatedAt, &user.UpdatedAt, &user.DeletedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *repository) IncreaseViewUpdates(ctx context.Context, uid string) (*User, error) {
	query := "UPDATE users SET view_updates = view_updates + 1 WHERE id = $1"
	row := r.db.QueryRowContext(ctx, query, uid)

	var user User
	err := row.Scan(&user.Id, &user.Username, &user.PasswordHash, &user.IsAcceptTerms, &user.IsAcceptPrivacy, &user.Is18, &user.TelegramChatId, &user.TelegramLinkedAt, &user.TelegramNotifications, &user.TokenVersion, &user.ViewUpdates, &user.ViewTutorial, &user.LoginedAt, &user.CreatedAt, &user.UpdatedAt, &user.DeletedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *repository) DeleteUser(ctx context.Context, id string) (*User, error) {
	logger.Debug(id)

	query := "DELETE FROM users WHERE id = $1"
	row := r.db.QueryRowContext(ctx, query, id)

	var user User
	err := row.Scan(&user.Id, &user.Username, &user.PasswordHash, &user.IsAcceptTerms, &user.IsAcceptPrivacy, &user.Is18, &user.TelegramChatId, &user.TelegramLinkedAt, &user.TelegramNotifications, &user.TokenVersion, &user.ViewUpdates, &user.ViewTutorial, &user.LoginedAt, &user.CreatedAt, &user.UpdatedAt, &user.DeletedAt)

	logger.Debug(user.Username)

	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &user, nil
}
