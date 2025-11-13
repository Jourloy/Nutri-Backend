package admin

import "time"

// AdminNotification представляет структуру рассылки от администратора
type AdminNotification struct {
	Id             int64      `json:"id" db:"id"`
	Title          string     `json:"title" db:"title"`
	Message        string     `json:"message" db:"message"`
	TargetAudience string     `json:"target_audience" db:"target_audience"`
	TargetPlanId   *int64     `json:"target_plan_id,omitempty" db:"target_plan_id"`
	TargetUserIds  []string   `json:"target_user_ids,omitempty" db:"target_user_ids"`
	Status         string     `json:"status" db:"status"`
	ScheduledAt    *time.Time `json:"scheduled_at,omitempty" db:"scheduled_at"`
	SentAt         *time.Time `json:"sent_at,omitempty" db:"sent_at"`
	CreatedBy      string     `json:"created_by" db:"created_by"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
}

// AdminNotificationCreate представляет структуру для создания рассылки
type AdminNotificationCreate struct {
	Title          string     `json:"title" db:"title"`
	Message        string     `json:"message" db:"message"`
	TargetAudience string     `json:"target_audience" db:"target_audience"` // 'all', 'free', 'premium', 'specific'
	TargetPlanId   *int64     `json:"target_plan_id,omitempty" db:"target_plan_id"`
	TargetUserIds  []string   `json:"target_user_ids,omitempty" db:"target_user_ids"`
	ScheduledAt    *time.Time `json:"scheduled_at,omitempty" db:"scheduled_at"`
}

// DashboardStats представляет статистику для дашборда
type DashboardStats struct {
	TotalUsers       int64   `json:"total_users"`
	ActiveUsers      int64   `json:"active_users"`
	FreeUsers        int64   `json:"free_users"`
	PremiumUsers     int64   `json:"premium_users"`
	MonthlyRevenue   int64   `json:"monthly_revenue"`
	NewUsersToday    int64   `json:"new_users_today"`
	NewUsersThisWeek int64   `json:"new_users_this_week"`
	ChurnRate        float64 `json:"churn_rate"`
}

// UserListItem представляет элемент в списке пользователей
type UserListItem struct {
	Id           string     `json:"id" db:"id"`
	Username     string     `json:"username" db:"username"`
	Email        *string    `json:"email,omitempty" db:"email"`
	Locale       string     `json:"locale" db:"locale"`
	PlanCode     *string    `json:"plan_code,omitempty" db:"plan_code"`
	PlanName     *string    `json:"plan_name,omitempty" db:"plan_name"`
	SubStatus    *string    `json:"sub_status,omitempty" db:"sub_status"`
	SubPeriodEnd *time.Time `json:"sub_period_end,omitempty" db:"sub_period_end"`
	LoginedAt    *time.Time `json:"logined_at,omitempty" db:"logined_at"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
}

// UserWithSubscription представляет пользователя с подпиской для создания
type UserWithSubscription struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	Email      string `json:"email,omitempty"`
	PlanId     int64  `json:"plan_id"`
	DurationMs int64  `json:"duration_ms"` // Длительность в миллисекундах
}

// UpdatePlanPrice представляет обновление цены тарифа
type UpdatePlanPrice struct {
	AmountMinor int64 `json:"amount_minor"`
}

// UpdateUserPlanPrice представляет обновление цены тарифа для конкретного пользователя
type UpdateUserPlanPrice struct {
	UserId      string `json:"user_id"`
	AmountMinor int64  `json:"amount_minor"`
}

// PlanWithFeatures представляет тариф с его возможностями
type PlanWithFeatures struct {
	Id            int64                  `json:"id"`
	Code          string                 `json:"code"`
	Name          string                 `json:"name"`
	AmountMinor   int64                  `json:"amount_minor"`
	Currency      string                 `json:"currency"`
	BillingPeriod string                 `json:"billing_period"`
	TrialDays     int                    `json:"trial_days"`
	IsActive      bool                   `json:"is_active"`
	Features      map[string]interface{} `json:"features"`
}
