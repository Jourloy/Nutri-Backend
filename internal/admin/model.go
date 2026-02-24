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
	Locale       *string    `json:"locale" db:"locale"`
	PlanCode     *string    `json:"plan_code,omitempty" db:"plan_code"`
	PlanName     *string    `json:"plan_name,omitempty" db:"plan_name"`
	SubStatus    *string    `json:"sub_status,omitempty" db:"sub_status"`
	SubPeriodEnd *time.Time `json:"sub_period_end,omitempty" db:"sub_period_end"`
	LoginedAt    *time.Time `json:"logined_at,omitempty" db:"logined_at"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
}

type UserSortBy string

const (
	UserSortByID           UserSortBy = "id"
	UserSortByUsername     UserSortBy = "username"
	UserSortByEmail        UserSortBy = "email"
	UserSortByLocale       UserSortBy = "locale"
	UserSortByPlanName     UserSortBy = "plan_name"
	UserSortBySubStatus    UserSortBy = "sub_status"
	UserSortBySubPeriodEnd UserSortBy = "sub_period_end"
	UserSortByLoginedAt    UserSortBy = "logined_at"
	UserSortByCreatedAt    UserSortBy = "created_at"
)

type SortOrder string

const (
	SortOrderAsc  SortOrder = "asc"
	SortOrderDesc SortOrder = "desc"
)

type AdminUserProfile struct {
	Id              string     `json:"id" db:"id"`
	Username        string     `json:"username" db:"username"`
	Email           *string    `json:"email,omitempty" db:"email"`
	EmailVerified   bool       `json:"email_verified" db:"email_verified"`
	Locale          *string    `json:"locale,omitempty" db:"locale"`
	Timezone        *string    `json:"timezone,omitempty" db:"timezone"`
	IsAcceptTerms   bool       `json:"is_accept_terms" db:"is_accept_terms"`
	IsAcceptPrivacy bool       `json:"is_accept_privacy" db:"is_accept_privacy"`
	Is18            bool       `json:"is_18" db:"is_18"`
	IsAdmin         bool       `json:"is_admin" db:"is_admin"`
	ViewUpdates     int64      `json:"view_updates" db:"view_updates"`
	ViewTutorial    int64      `json:"view_tutorial" db:"view_tutorial"`
	LoginedAt       *time.Time `json:"logined_at,omitempty" db:"logined_at"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
}

type AdminUserSubscription struct {
	Id                   int64      `json:"id" db:"id"`
	UserId               string     `json:"user_id" db:"user_id"`
	PlanId               int64      `json:"plan_id" db:"plan_id"`
	PlanCode             *string    `json:"plan_code,omitempty" db:"plan_code"`
	PlanName             *string    `json:"plan_name,omitempty" db:"plan_name"`
	Status               string     `json:"status" db:"status"`
	PeriodStart          time.Time  `json:"period_start" db:"period_start"`
	PeriodEnd            time.Time  `json:"period_end" db:"period_end"`
	CancelAt             *time.Time `json:"cancel_at,omitempty" db:"cancel_at"`
	CanceledAt           *time.Time `json:"canceled_at,omitempty" db:"canceled_at"`
	TrialEnd             *time.Time `json:"trial_end,omitempty" db:"trial_end"`
	AmountMinor          int64      `json:"amount_minor" db:"amount_minor"`
	Currency             string     `json:"currency" db:"currency"`
	BillingPeriod        string     `json:"billing_period" db:"billing_period"`
	ExternalSubscription *string    `json:"external_subscription_id,omitempty" db:"external_subscription_id"`
	ExternalCustomer     *string    `json:"external_customer_id,omitempty" db:"external_customer_id"`
	AdCode               *string    `json:"ad_code,omitempty" db:"ad_code"`
	CreatedAt            time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at" db:"updated_at"`
}

type AdminUserSummary struct {
	Subscriptions int64 `json:"subscriptions" db:"subscriptions"`
	Orders        int64 `json:"orders" db:"orders"`
	Tickets       int64 `json:"tickets" db:"tickets"`
	Products      int64 `json:"products" db:"products"`
	Recipes       int64 `json:"recipes" db:"recipes"`
	Supplements   int64 `json:"supplements" db:"supplements"`
	Achievements  int64 `json:"achievements" db:"achievements"`
	Feedbacks     int64 `json:"feedbacks" db:"feedbacks"`
	BodyWeights   int64 `json:"body_weights" db:"body_weights"`
	BodyMeasures  int64 `json:"body_measurements" db:"body_measurements"`
	BodyActivity  int64 `json:"body_activity" db:"body_activity"`
	BodyWorkouts  int64 `json:"body_workouts" db:"body_workouts"`
	AIAnalysis    int64 `json:"ai_analysis_logs" db:"ai_analysis_logs"`
}

type AdminUserLatestOrder struct {
	Id          int64      `json:"id" db:"id"`
	Status      string     `json:"status" db:"status"`
	PlanId      int64      `json:"plan_id" db:"plan_id"`
	AmountMinor int64      `json:"amount_minor" db:"amount_minor"`
	Currency    string     `json:"currency" db:"currency"`
	Provider    string     `json:"provider" db:"provider"`
	PaidAt      *time.Time `json:"paid_at,omitempty" db:"paid_at"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

type AdminUserLatestTicket struct {
	Id        int64      `json:"id" db:"id"`
	Subject   string     `json:"subject" db:"subject"`
	Status    string     `json:"status" db:"status"`
	Priority  string     `json:"priority" db:"priority"`
	Category  *string    `json:"category,omitempty" db:"category"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	ClosedAt  *time.Time `json:"closed_at,omitempty" db:"closed_at"`
}

type UserDetailsResponse struct {
	User                *AdminUserProfile      `json:"user"`
	CurrentSubscription *AdminUserSubscription `json:"current_subscription,omitempty"`
	Summary             AdminUserSummary       `json:"summary"`
	LatestOrder         *AdminUserLatestOrder  `json:"latest_order,omitempty"`
	LatestTicket        *AdminUserLatestTicket `json:"latest_ticket,omitempty"`
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

type GrantUserSubscriptionRequest struct {
	PlanId       int64 `json:"plan_id"`
	DurationDays int64 `json:"duration_days"`
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
