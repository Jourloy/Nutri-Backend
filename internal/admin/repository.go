package admin

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/jourloy/nutri02/internal/database"
)

type Repository interface {
	// Dashboard
	GetDashboardStats(ctx context.Context) (*DashboardStats, error)

	// Users
	GetAllUsers(ctx context.Context, limit, offset int, sortBy UserSortBy, sortOrder SortOrder) ([]UserListItem, error)
	GetUserCount(ctx context.Context) (int64, error)
	GetUserDetails(ctx context.Context, userId string) (*UserDetailsResponse, error)
	CreateUserWithSubscription(ctx context.Context, username, passwordHash, email string, planId int64, durationMs int64) error
	GrantUserSubscription(ctx context.Context, userId string, planId, durationDays int64) (*AdminUserSubscription, error)

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

func resolveUserSort(sortBy UserSortBy, sortOrder SortOrder) (string, string) {
	column := "u.created_at"
	switch sortBy {
	case UserSortByID:
		column = "u.id"
	case UserSortByUsername:
		column = "u.username"
	case UserSortByEmail:
		column = "u.email"
	case UserSortByLocale:
		column = "u.locale"
	case UserSortByPlanName:
		column = "p.name"
	case UserSortBySubStatus:
		column = "s.status"
	case UserSortBySubPeriodEnd:
		column = "s.period_end"
	case UserSortByLoginedAt:
		column = "u.logined_at"
	case UserSortByCreatedAt:
		column = "u.created_at"
	}

	direction := "DESC"
	if sortOrder == SortOrderAsc {
		direction = "ASC"
	}

	return column, direction
}

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

// GetAllUsers получает список всех пользователей с пагинацией и сортировкой
func (r *repository) GetAllUsers(ctx context.Context, limit, offset int, sortBy UserSortBy, sortOrder SortOrder) ([]UserListItem, error) {
	sortColumn, sortDirection := resolveUserSort(sortBy, sortOrder)

	q := fmt.Sprintf(`
			SELECT
				u.id, u.username, u.email, u.locale, u.logined_at, u.created_at,
				p.code as plan_code, p.name as plan_name,
				s.status as sub_status, s.period_end as sub_period_end
			FROM users u
			LEFT JOIN subscriptions s ON u.id = s.user_id AND s.status IN ('active', 'trialing', 'past_due')
			LEFT JOIN plans p ON s.plan_id = p.id
			WHERE u.deleted_at IS NULL
			ORDER BY %s %s NULLS LAST, u.id ASC
			LIMIT $1 OFFSET $2
		`, sortColumn, sortDirection)

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

func (r *repository) GetUserDetails(ctx context.Context, userId string) (*UserDetailsResponse, error) {
	const userQ = `
		SELECT
			id, username, email, email_verified, locale, timezone,
			is_accept_terms, is_accept_privacy, is_18, is_admin,
			view_updates, view_tutorial, logined_at, created_at, updated_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`

	var profile AdminUserProfile
	if err := r.db.GetContext(ctx, &profile, userQ, userId); err != nil {
		return nil, err
	}

	resp := &UserDetailsResponse{
		User: &profile,
	}

	const subQ = `
		SELECT
			s.id, s.user_id, s.plan_id,
			p.code AS plan_code, p.name AS plan_name,
			s.status, s.period_start, s.period_end,
			s.cancel_at, s.canceled_at, s.trial_end,
			s.amount_minor, s.currency, s.billing_period,
			s.external_subscription_id, s.external_customer_id,
			s.ad_code, s.created_at, s.updated_at
		FROM subscriptions s
		JOIN plans p ON p.id = s.plan_id
		WHERE s.user_id = $1
		ORDER BY
			CASE WHEN s.status IN ('active', 'trialing', 'past_due') THEN 0 ELSE 1 END,
			s.updated_at DESC,
			s.id DESC
		LIMIT 1
	`

	var sub AdminUserSubscription
	if err := r.db.GetContext(ctx, &sub, subQ, userId); err != nil {
		if err != sql.ErrNoRows {
			return nil, err
		}
	} else {
		resp.CurrentSubscription = &sub
	}

	const summaryQ = `
		SELECT
			(SELECT COUNT(*) FROM subscriptions WHERE user_id = $1) AS subscriptions,
			(SELECT COUNT(*) FROM orders WHERE user_id = $1) AS orders,
			(SELECT COUNT(*) FROM tickets WHERE user_id = $1) AS tickets,
			(SELECT COUNT(*) FROM products WHERE user_id = $1) AS products,
			(SELECT COUNT(*) FROM recipes WHERE user_id = $1 AND deleted_at IS NULL) AS recipes,
			(SELECT COUNT(*) FROM supplements WHERE user_id = $1 AND deleted_at IS NULL) AS supplements,
			(SELECT COUNT(*) FROM user_achievements WHERE user_id = $1) AS achievements,
			(SELECT COUNT(*) FROM feedbacks WHERE user_id = $1) AS feedbacks,
			(SELECT COUNT(*) FROM body_weights WHERE user_id = $1) AS body_weights,
			(SELECT COUNT(*) FROM body_measurements WHERE user_id = $1) AS body_measurements,
			(SELECT COUNT(*) FROM body_activity WHERE user_id = $1) AS body_activity,
			(SELECT COUNT(*) FROM body_workouts WHERE user_id = $1) AS body_workouts,
			(SELECT COUNT(*) FROM ai_analysis_logs WHERE user_id = $1) AS ai_analysis_logs
	`
	if err := r.db.GetContext(ctx, &resp.Summary, summaryQ, userId); err != nil {
		return nil, err
	}

	const orderQ = `
		SELECT
			id, status, plan_id, amount_minor, currency, provider,
			paid_at, created_at, updated_at
		FROM orders
		WHERE user_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT 1
	`

	var order AdminUserLatestOrder
	if err := r.db.GetContext(ctx, &order, orderQ, userId); err != nil {
		if err != sql.ErrNoRows {
			return nil, err
		}
	} else {
		resp.LatestOrder = &order
	}

	const ticketQ = `
		SELECT
			id, subject, status, priority, category,
			created_at, updated_at, closed_at
		FROM tickets
		WHERE user_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT 1
	`

	var ticket AdminUserLatestTicket
	if err := r.db.GetContext(ctx, &ticket, ticketQ, userId); err != nil {
		if err != sql.ErrNoRows {
			return nil, err
		}
	} else {
		resp.LatestTicket = &ticket
	}

	return resp, nil
}

func (r *repository) GrantUserSubscription(ctx context.Context, userId string, planId, durationDays int64) (*AdminUserSubscription, error) {
	if durationDays <= 0 {
		return nil, fmt.Errorf("duration_days must be greater than zero")
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	const userExistsQ = `
		SELECT id
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`
	var existingUserID string
	if err := tx.GetContext(ctx, &existingUserID, userExistsQ, userId); err != nil {
		return nil, err
	}

	const planQ = `
		SELECT id, amount_minor, currency, billing_period
		FROM plans
		WHERE id = $1
	`
	var plan struct {
		Id            int64  `db:"id"`
		AmountMinor   int64  `db:"amount_minor"`
		Currency      string `db:"currency"`
		BillingPeriod string `db:"billing_period"`
	}
	if err := tx.GetContext(ctx, &plan, planQ, planId); err != nil {
		return nil, err
	}

	now := time.Now()
	periodEnd := now.Add(time.Duration(durationDays) * 24 * time.Hour)

	var targetSubID int64
	const activeSubQ = `
		SELECT id
		FROM subscriptions
		WHERE user_id = $1 AND status IN ('active', 'trialing', 'past_due')
		ORDER BY updated_at DESC, id DESC
		LIMIT 1
		FOR UPDATE
	`
	err = tx.GetContext(ctx, &targetSubID, activeSubQ, userId)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	if err == sql.ErrNoRows {
		const latestSubQ = `
			SELECT id
			FROM subscriptions
			WHERE user_id = $1
			ORDER BY updated_at DESC, id DESC
			LIMIT 1
			FOR UPDATE
		`
		err = tx.GetContext(ctx, &targetSubID, latestSubQ, userId)
		if err != nil && err != sql.ErrNoRows {
			return nil, err
		}
	}

	if targetSubID > 0 {
		const updateSubQ = `
			UPDATE subscriptions
			SET
				plan_id = $2,
				status = 'active',
				period_start = $3,
				period_end = $4,
				cancel_at = NULL,
				canceled_at = NULL,
				trial_end = NULL,
				amount_minor = $5,
				currency = $6,
				billing_period = $7,
				external_subscription_id = NULL,
				external_customer_id = NULL,
				updated_at = NOW()
			WHERE id = $1
		`
		if _, err := tx.ExecContext(ctx, updateSubQ, targetSubID, plan.Id, now, periodEnd, plan.AmountMinor, plan.Currency, plan.BillingPeriod); err != nil {
			return nil, err
		}
	} else {
		const insertSubQ = `
			INSERT INTO subscriptions (
				user_id, plan_id, status, period_start, period_end,
				cancel_at, canceled_at, trial_end,
				amount_minor, currency, billing_period,
				external_subscription_id, external_customer_id
			) VALUES (
				$1, $2, 'active', $3, $4,
				NULL, NULL, NULL,
				$5, $6, $7,
				NULL, NULL
			)
			RETURNING id
		`
		if err := tx.GetContext(ctx, &targetSubID, insertSubQ, userId, plan.Id, now, periodEnd, plan.AmountMinor, plan.Currency, plan.BillingPeriod); err != nil {
			return nil, err
		}
	}

	const selectSubQ = `
		SELECT
			s.id, s.user_id, s.plan_id,
			p.code AS plan_code, p.name AS plan_name,
			s.status, s.period_start, s.period_end,
			s.cancel_at, s.canceled_at, s.trial_end,
			s.amount_minor, s.currency, s.billing_period,
			s.external_subscription_id, s.external_customer_id,
			s.ad_code, s.created_at, s.updated_at
		FROM subscriptions s
		JOIN plans p ON p.id = s.plan_id
		WHERE s.id = $1
	`
	var sub AdminUserSubscription
	if err := tx.GetContext(ctx, &sub, selectSubQ, targetSubID); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &sub, nil
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
