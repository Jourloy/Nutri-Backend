package telegram

import (
    "context"

    "github.com/jmoiron/sqlx"

    "github.com/jourloy/nutri-backend/internal/database"
)

type Repository interface {
    CreateOrGetTicket(ctx context.Context, userId string) (*TelegramProfile, error)
    LinkByToken(ctx context.Context, in LinkRequest) (*TelegramProfile, error)
    GetByUserId(ctx context.Context, userId string) (*TelegramProfile, error)
    GetByTelegramId(ctx context.Context, telegramId string) (*TelegramProfile, error)
    GetPublicByUserId(ctx context.Context, userId string) (*TelegramPublic, error)
    DeleteByUserId(ctx context.Context, userId string) error
    UpdateNotifyByUserId(ctx context.Context, userId string, upd NotifyUpdate) (*TelegramProfile, error)
    GetAllForDailyNotify(ctx context.Context) ([]TelegramProfileWithUser, error)
    UpdateLastDailyReminder(ctx context.Context, telegramId string) error
}

type repository struct {
    db *sqlx.DB
}

func NewRepository() Repository {
    return &repository{db: database.Database}
}

const columns = `
    id, token, telegram_id, telegram_username, telegram_avatar,
    notify_daily, notify_story, user_id, connected_at, created_at, updated_at
`

// CreateOrGetTicket creates a profile for the user if it doesn't exist and returns it (with token).
func (r *repository) CreateOrGetTicket(ctx context.Context, userId string) (*TelegramProfile, error) {
    // Single statement: insert if absent, else return existing.
    const q = `
        WITH ins AS (
            INSERT INTO telegram_profiles (user_id)
            SELECT $1
            WHERE NOT EXISTS (
                SELECT 1 FROM telegram_profiles WHERE user_id = $1
            )
            RETURNING ` + columns + `
        )
        SELECT ` + columns + ` FROM ins
        UNION ALL
        SELECT ` + columns + ` FROM telegram_profiles WHERE user_id = $1
        LIMIT 1;`

    var tp TelegramProfile
    if err := r.db.GetContext(ctx, &tp, q, userId); err != nil {
        return nil, err
    }
    return &tp, nil
}

// LinkByToken fills telegram_* fields and connected_at using a one-time token.
func (r *repository) LinkByToken(ctx context.Context, in LinkRequest) (*TelegramProfile, error) {
    const q = `
        UPDATE telegram_profiles
        SET telegram_id = $2,
            telegram_username = $3,
            telegram_avatar = $4,
            connected_at = NOW(),
            updated_at = NOW()
        WHERE token = $1
        RETURNING ` + columns + `;`

    var tp TelegramProfile
    if err := r.db.GetContext(ctx, &tp, q, in.Token, in.TelegramId, in.TelegramUsername, in.TelegramAvatar); err != nil {
        return nil, err
    }
    return &tp, nil
}

func (r *repository) GetByUserId(ctx context.Context, userId string) (*TelegramProfile, error) {
    const q = `SELECT ` + columns + ` FROM telegram_profiles WHERE user_id = $1 LIMIT 1;`
    var tp TelegramProfile
    if err := r.db.GetContext(ctx, &tp, q, userId); err != nil {
        return nil, err
    }
    return &tp, nil
}

func (r *repository) GetPublicByUserId(ctx context.Context, userId string) (*TelegramPublic, error) {
    const q = `
        SELECT telegram_id, telegram_username, telegram_avatar
        FROM telegram_profiles
        WHERE user_id = $1
        LIMIT 1;`
    var pub TelegramPublic
    if err := r.db.GetContext(ctx, &pub, q, userId); err != nil {
        return nil, err
    }
    return &pub, nil
}

func (r *repository) DeleteByUserId(ctx context.Context, userId string) error {
    const q = `DELETE FROM telegram_profiles WHERE user_id = $1 RETURNING id;`
    var id int64
    if err := r.db.GetContext(ctx, &id, q, userId); err != nil {
        return err
    }
    return nil
}

func (r *repository) UpdateNotifyByUserId(ctx context.Context, userId string, upd NotifyUpdate) (*TelegramProfile, error) {
    const q = `
        UPDATE telegram_profiles
        SET notify_daily = COALESCE($2, notify_daily),
            notify_story = COALESCE($3, notify_story),
            updated_at   = NOW()
        WHERE user_id = $1
        RETURNING ` + columns + `;`

    var tp TelegramProfile
    if err := r.db.GetContext(ctx, &tp, q, userId, upd.NotifyDaily, upd.NotifyStory); err != nil {
        return nil, err
    }
    return &tp, nil
}

// GetByTelegramId returns telegram profile by telegram_id
func (r *repository) GetByTelegramId(ctx context.Context, telegramId string) (*TelegramProfile, error) {
    const q = `SELECT ` + columns + ` FROM telegram_profiles WHERE telegram_id = $1 LIMIT 1;`
    var tp TelegramProfile
    if err := r.db.GetContext(ctx, &tp, q, telegramId); err != nil {
        return nil, err
    }
    return &tp, nil
}

// GetAllForDailyNotify returns all profiles with notify_daily=true and telegram_id set
func (r *repository) GetAllForDailyNotify(ctx context.Context) ([]TelegramProfileWithUser, error) {
    const q = `
        SELECT
            tp.id, tp.token, tp.telegram_id, tp.telegram_username, tp.telegram_avatar,
            tp.notify_daily, tp.notify_story, tp.user_id, tp.connected_at,
            tp.created_at, tp.updated_at, tp.last_daily_reminder_at,
            u.timezone as user_timezone, u.locale as user_locale
        FROM telegram_profiles tp
        JOIN users u ON tp.user_id = u.id
        WHERE tp.notify_daily = true
          AND tp.telegram_id IS NOT NULL
          AND u.deleted_at IS NULL;`

    var profiles []TelegramProfileWithUser
    if err := r.db.SelectContext(ctx, &profiles, q); err != nil {
        return nil, err
    }
    return profiles, nil
}

// UpdateLastDailyReminder updates the last_daily_reminder_at timestamp
func (r *repository) UpdateLastDailyReminder(ctx context.Context, telegramId string) error {
    const q = `
        UPDATE telegram_profiles
        SET last_daily_reminder_at = NOW(),
            updated_at = NOW()
        WHERE telegram_id = $1
        RETURNING id;`

    var id int64
    if err := r.db.GetContext(ctx, &id, q, telegramId); err != nil {
        return err
    }
    return nil
}
