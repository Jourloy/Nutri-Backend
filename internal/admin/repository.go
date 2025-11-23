package admin

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/jourloy/nutri-backend/internal/database"
)

type Repository interface {
	// Dashboard
	GetDashboardStats(ctx context.Context) (*DashboardStats, error)

	// Users
	GetAllUsers(ctx context.Context, limit, offset int) ([]UserListItem, error)
	GetUserCount(ctx context.Context) (int64, error)
	CreateUserWithSubscription(ctx context.Context, username, passwordHash, email string, planId int64, durationMs int64) error

	// Plans
	UpdatePlanPrice(ctx context.Context, planId int64, amountMinor int64) error
	UpdateUserSubscriptionPrice(ctx context.Context, userId string, amountMinor int64) error
	UpdatePlanFeatures(ctx context.Context, planId int64, features map[string]interface{}) error

	// Notifications
	CreateNotification(ctx context.Context, createdBy string, notification *AdminNotificationCreate) (*AdminNotification, error)
	GetNotifications(ctx context.Context, limit, offset int) ([]AdminNotification, error)
	SendNotification(ctx context.Context, notificationId int64) error
}

type repository struct {
	db *sqlx.DB
}

func NewRepository() Repository {
	return &repository{db: database.Database}
}

const notificationColumns = `
	id, title, message, target_audience, target_plan_id, target_user_ids,
	status, scheduled_at, sent_at, created_by, created_at, updated_at
`

// GetDashboardStats получает статистику для дашборда
func (r *repository) GetDashboardStats(ctx context.Context) (*DashboardStats, error) {
	stats := &DashboardStats{}

	// Общее количество пользователей
	const totalUsersQ = `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`
	if err := r.db.GetContext(ctx, &stats.TotalUsers, totalUsersQ); err != nil {
		return nil, err
	}

	// Активные пользователи (заходили за последние 3 дней)
	const activeUsersQ = `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL AND logined_at > NOW() - INTERVAL '3 days'`
	if err := r.db.GetContext(ctx, &stats.ActiveUsers, activeUsersQ); err != nil {
		return nil, err
	}

	// Пользователи с бесплатным тарифом
	const freeUsersQ = `
		SELECT COUNT(DISTINCT u.id)
		FROM users u
		LEFT JOIN subscriptions s ON u.id = s.user_id AND s.status IN ('active', 'trialing')
		LEFT JOIN plans p ON s.plan_id = p.id
		WHERE u.deleted_at IS NULL AND (p.code = 'START' OR p.code IS NULL)
	`
	if err := r.db.GetContext(ctx, &stats.FreeUsers, freeUsersQ); err != nil {
		return nil, err
	}

	// Пользователи с платным тарифом
	const premiumUsersQ = `
		SELECT COUNT(DISTINCT u.id)
		FROM users u
		JOIN subscriptions s ON u.id = s.user_id AND s.status IN ('active', 'trialing')
		JOIN plans p ON s.plan_id = p.id
		WHERE u.deleted_at IS NULL AND p.code != 'START'
	`
	if err := r.db.GetContext(ctx, &stats.PremiumUsers, premiumUsersQ); err != nil {
		return nil, err
	}

	// Месячный доход (сумма всех активных подписок)
	const monthlyRevenueQ = `
		SELECT COALESCE(SUM(s.amount_minor), 0)
		FROM subscriptions s
		JOIN plans p ON s.plan_id = p.id
		WHERE s.status IN ('active', 'trialing') AND s.billing_period = 'month' AND p.code != 'START'
	`
	if err := r.db.GetContext(ctx, &stats.MonthlyRevenue, monthlyRevenueQ); err != nil {
		return nil, err
	}

	// Новые пользователи сегодня
	const newUsersTodayQ = `SELECT COUNT(*) FROM users WHERE created_at::date = CURRENT_DATE`
	if err := r.db.GetContext(ctx, &stats.NewUsersToday, newUsersTodayQ); err != nil {
		return nil, err
	}

	// Новые пользователи за неделю
	const newUsersWeekQ = `SELECT COUNT(*) FROM users WHERE created_at > NOW() - INTERVAL '7 days'`
	if err := r.db.GetContext(ctx, &stats.NewUsersThisWeek, newUsersWeekQ); err != nil {
		return nil, err
	}

	// Churn rate (процент отмененных подписок за последний месяц)
	const churnRateQ = `
		SELECT
			CASE
				WHEN COUNT(*) = 0 THEN 0
				ELSE (COUNT(*) FILTER (WHERE status = 'canceled' AND canceled_at > NOW() - INTERVAL '30 days'))::FLOAT / COUNT(*)::FLOAT * 100
			END
		FROM subscriptions
	`
	if err := r.db.GetContext(ctx, &stats.ChurnRate, churnRateQ); err != nil {
		return nil, err
	}

	return stats, nil
}

// GetAllUsers получает список всех пользователей с пагинацией
func (r *repository) GetAllUsers(ctx context.Context, limit, offset int) ([]UserListItem, error) {
	const q = `
		SELECT
			u.id, u.username, u.email, u.locale, u.logined_at, u.created_at,
			p.code as plan_code, p.name as plan_name,
			s.status as sub_status, s.period_end as sub_period_end
		FROM users u
		LEFT JOIN subscriptions s ON u.id = s.user_id AND s.status IN ('active', 'trialing')
		LEFT JOIN plans p ON s.plan_id = p.id
		WHERE u.deleted_at IS NULL
		ORDER BY u.created_at DESC
		LIMIT $1 OFFSET $2
	`

	var users []UserListItem
	if err := r.db.SelectContext(ctx, &users, q, limit, offset); err != nil {
		return nil, err
	}

	return users, nil
}

// GetUserCount получает общее количество пользователей
func (r *repository) GetUserCount(ctx context.Context) (int64, error) {
	const q = `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`
	var count int64
	if err := r.db.GetContext(ctx, &count, q); err != nil {
		return 0, err
	}
	return count, nil
}

// CreateUserWithSubscription создает пользователя с подпиской
func (r *repository) CreateUserWithSubscription(ctx context.Context, username, passwordHash, email string, planId int64, durationMs int64) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Создаем пользователя
	const userQ = `
		INSERT INTO users (username, password_hash, email)
		VALUES ($1, $2, NULLIF($3, ''))
		RETURNING id
	`
	var userId string
	if err := tx.GetContext(ctx, &userId, userQ, username, passwordHash, email); err != nil {
		return err
	}

	// Получаем информацию о тарифе
	const planQ = `SELECT amount_minor, currency, billing_period FROM plans WHERE id = $1`
	var plan struct {
		AmountMinor   int64  `db:"amount_minor"`
		Currency      string `db:"currency"`
		BillingPeriod string `db:"billing_period"`
	}
	if err := tx.GetContext(ctx, &plan, planQ, planId); err != nil {
		return err
	}

	// Создаем подписку
	periodStart := time.Now()
	periodEnd := periodStart.Add(time.Duration(durationMs) * time.Millisecond)

	const subQ = `
		INSERT INTO subscriptions (
			user_id, plan_id, status, period_start, period_end,
			amount_minor, currency, billing_period
		) VALUES ($1, $2, 'active', $3, $4, $5, $6, $7)
	`
	if _, err := tx.ExecContext(ctx, subQ, userId, planId, periodStart, periodEnd, plan.AmountMinor, plan.Currency, plan.BillingPeriod); err != nil {
		return err
	}

	return tx.Commit()
}

// UpdatePlanPrice обновляет цену тарифа для всех (будущих) подписок
func (r *repository) UpdatePlanPrice(ctx context.Context, planId int64, amountMinor int64) error {
	const q = `
		UPDATE plans
		SET amount_minor = $2, updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, q, planId, amountMinor)
	return err
}

// UpdateUserSubscriptionPrice обновляет цену подписки для конкретного пользователя
func (r *repository) UpdateUserSubscriptionPrice(ctx context.Context, userId string, amountMinor int64) error {
	const q = `
		UPDATE subscriptions
		SET amount_minor = $2, updated_at = NOW()
		WHERE user_id = $1 AND status IN ('active', 'trialing')
	`
	_, err := r.db.ExecContext(ctx, q, userId, amountMinor)
	return err
}

// UpdatePlanFeatures обновляет возможности тарифа
func (r *repository) UpdatePlanFeatures(ctx context.Context, planId int64, features map[string]interface{}) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Удаляем старые фичи
	const deleteQ = `DELETE FROM plan_features WHERE plan_id = $1`
	if _, err := tx.ExecContext(ctx, deleteQ, planId); err != nil {
		return err
	}

	// Добавляем новые фичи
	const insertQ = `
		INSERT INTO plan_features (plan_id, feature_key, value)
		VALUES ($1, $2, $3)
	`
	for key, value := range features {
		if _, err := tx.ExecContext(ctx, insertQ, planId, key, value); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// CreateNotification создает новое уведомление
func (r *repository) CreateNotification(ctx context.Context, createdBy string, notification *AdminNotificationCreate) (*AdminNotification, error) {
	const q = `
		INSERT INTO admin_notifications (
			title, message, target_audience, target_plan_id, target_user_ids,
			status, scheduled_at, created_by
		) VALUES (
			:title, :message, :target_audience, :target_plan_id, :target_user_ids,
			CASE WHEN :scheduled_at IS NULL THEN 'draft' ELSE 'scheduled' END,
			:scheduled_at, :created_by
		)
		RETURNING ` + notificationColumns

	args := map[string]interface{}{
		"title":           notification.Title,
		"message":         notification.Message,
		"target_audience": notification.TargetAudience,
		"target_plan_id":  notification.TargetPlanId,
		"target_user_ids": pq.Array(notification.TargetUserIds),
		"scheduled_at":    notification.ScheduledAt,
		"created_by":      createdBy,
	}

	rows, err := r.db.NamedQueryContext(ctx, q, args)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var n AdminNotification
		if err := rows.StructScan(&n); err != nil {
			return nil, err
		}
		return &n, nil
	}

	return nil, sql.ErrNoRows
}

// GetNotifications получает список уведомлений
func (r *repository) GetNotifications(ctx context.Context, limit, offset int) ([]AdminNotification, error) {
	const q = `
		SELECT ` + notificationColumns + `
		FROM admin_notifications
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	var notifications []AdminNotification
	if err := r.db.SelectContext(ctx, &notifications, q, limit, offset); err != nil {
		return nil, err
	}

	return notifications, nil
}

// SendNotification отправляет уведомление (создает записи доставки)
func (r *repository) SendNotification(ctx context.Context, notificationId int64) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Получаем информацию об уведомлении
	const notifQ = `SELECT target_audience, target_plan_id, target_user_ids FROM admin_notifications WHERE id = $1`
	var notif struct {
		TargetAudience string   `db:"target_audience"`
		TargetPlanId   *int64   `db:"target_plan_id"`
		TargetUserIds  []string `db:"target_user_ids"`
	}
	if err := tx.GetContext(ctx, &notif, notifQ, notificationId); err != nil {
		return err
	}

	var userIds []string

	// Определяем целевых пользователей
	switch notif.TargetAudience {
	case "all":
		const allQ = `SELECT id FROM users WHERE deleted_at IS NULL`
		if err := tx.SelectContext(ctx, &userIds, allQ); err != nil {
			return err
		}

	case "free":
		const freeQ = `
			SELECT DISTINCT u.id
			FROM users u
			LEFT JOIN subscriptions s ON u.id = s.user_id AND s.status IN ('active', 'trialing')
			LEFT JOIN plans p ON s.plan_id = p.id
			WHERE u.deleted_at IS NULL AND (p.code = 'FREE' OR p.code IS NULL)
		`
		if err := tx.SelectContext(ctx, &userIds, freeQ); err != nil {
			return err
		}

	case "premium":
		const premiumQ = `
			SELECT DISTINCT u.id
			FROM users u
			JOIN subscriptions s ON u.id = s.user_id AND s.status IN ('active', 'trialing')
			JOIN plans p ON s.plan_id = p.id
			WHERE u.deleted_at IS NULL AND p.code != 'FREE'
		`
		if err := tx.SelectContext(ctx, &userIds, premiumQ); err != nil {
			return err
		}

	case "plan":
		if notif.TargetPlanId == nil {
			return sql.ErrNoRows
		}
		const planQ = `
			SELECT DISTINCT u.id
			FROM users u
			JOIN subscriptions s ON u.id = s.user_id AND s.status IN ('active', 'trialing')
			WHERE u.deleted_at IS NULL AND s.plan_id = $1
		`
		if err := tx.SelectContext(ctx, &userIds, planQ, *notif.TargetPlanId); err != nil {
			return err
		}

	case "specific":
		userIds = notif.TargetUserIds
	}

	// Создаем записи доставки
	const deliveryQ = `
		INSERT INTO notification_deliveries (notification_id, user_id)
		VALUES ($1, $2)
	`
	for _, userId := range userIds {
		if _, err := tx.ExecContext(ctx, deliveryQ, notificationId, userId); err != nil {
			return err
		}
	}

	// Обновляем статус уведомления
	const updateQ = `
		UPDATE admin_notifications
		SET status = 'sent', sent_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`
	if _, err := tx.ExecContext(ctx, updateQ, notificationId); err != nil {
		return err
	}

	return tx.Commit()
}
