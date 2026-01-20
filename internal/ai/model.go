package ai

import "time"

// AnalysisLog represents a log entry for AI analysis requests
type AnalysisLog struct {
	Id          int64    `json:"id" db:"id"`
	UserId      string   `json:"userId" db:"user_id"`
	RequestType string   `json:"requestType" db:"request_type"`
	ImageUrl    string   `json:"imageUrl,omitempty" db:"image_url"`
	UserPrompt  string   `json:"userPrompt,omitempty" db:"user_prompt"`
	TotalWeight *float64 `json:"totalWeight,omitempty" db:"total_weight"`

	ResponseData *string `json:"responseData,omitempty" db:"response_data"`
	ParsedResult *string `json:"parsedResult,omitempty" db:"parsed_result"`

	ModelUsed        string   `json:"modelUsed" db:"model_used"`
	TokensPrompt     *int     `json:"tokensPrompt,omitempty" db:"tokens_prompt"`
	TokensCompletion *int     `json:"tokensCompletion,omitempty" db:"tokens_completion"`
	EstimatedCostUsd *float64 `json:"estimatedCostUsd,omitempty" db:"estimated_cost_usd"`

	Status           string  `json:"status" db:"status"`
	ErrorMessage     *string `json:"errorMessage,omitempty" db:"error_message"`
	ProcessingTimeMs *int    `json:"processingTimeMs,omitempty" db:"processing_time_ms"`

	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

// UserLimit represents daily usage limits for a user
type UserLimit struct {
	Id               int64     `json:"id" db:"id"`
	UserId           string    `json:"userId" db:"user_id"`
	LimitDate        string    `json:"limitDate" db:"limit_date"`
	RequestType      string    `json:"requestType" db:"request_type"`
	RequestsCount    int       `json:"requestsCount" db:"requests_count"`
	MaxRequests      int       `json:"maxRequests" db:"max_requests"`
	SubscriptionTier *string   `json:"subscriptionTier,omitempty" db:"subscription_tier"`
	CreatedAt        time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt        time.Time `json:"updatedAt" db:"updated_at"`
}

// Violation represents an AI usage violation
type Violation struct {
	Id              int64      `json:"id" db:"id"`
	UserId          string     `json:"userId" db:"user_id"`
	AnalysisLogId   *int64     `json:"analysisLogId,omitempty" db:"analysis_log_id"`
	ViolationType   string     `json:"violationType" db:"violation_type"`
	ViolationReason string     `json:"violationReason" db:"violation_reason"`
	ImageUrl        *string    `json:"imageUrl,omitempty" db:"image_url"`
	UserPrompt      *string    `json:"userPrompt,omitempty" db:"user_prompt"`
	ActionTaken     string     `json:"actionTaken" db:"action_taken"`
	BanUntil        *time.Time `json:"banUntil,omitempty" db:"ban_until"`
	Reviewed        bool       `json:"reviewed" db:"reviewed"`
	ReviewedBy      *string    `json:"reviewedBy,omitempty" db:"reviewed_by"`
	ReviewedAt      *time.Time `json:"reviewedAt,omitempty" db:"reviewed_at"`
	CreatedAt       time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt       time.Time  `json:"updatedAt" db:"updated_at"`
}

// AdminNotification represents a notification for admin panel
type AdminNotification struct {
	Id               int64      `json:"id" db:"id"`
	NotificationType string     `json:"notificationType" db:"notification_type"`
	Title            string     `json:"title" db:"title"`
	Message          string     `json:"message" db:"message"`
	Severity         string     `json:"severity" db:"severity"`
	UserId           *string    `json:"userId,omitempty" db:"user_id"`
	RelatedId        *int64     `json:"relatedId,omitempty" db:"related_id"`
	Metadata         *string    `json:"metadata,omitempty" db:"metadata"`
	Read             bool       `json:"read" db:"read"`
	ReadBy           *string    `json:"readBy,omitempty" db:"read_by"`
	ReadAt           *time.Time `json:"readAt,omitempty" db:"read_at"`
	CreatedAt        time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt        time.Time  `json:"updatedAt" db:"updated_at"`
}

// FoodAnalysisRequest represents the request for food analysis
type FoodAnalysisRequest struct {
	Image       []byte  `json:"-"` // Image data
	Description string  `json:"description"`
	TotalWeight float64 `json:"totalWeight"` // in grams
}

// FoodAnalysisResult represents the parsed result from AI
type FoodAnalysisResult struct {
	Calories           float64  `json:"calories"`
	Protein            float64  `json:"protein"`
	Fat                float64  `json:"fat"`
	Carbs              float64  `json:"carbs"`
	Fiber              *float64 `json:"fiber"`              // nullable
	Cholesterol        *float64 `json:"cholesterol"`        // nullable, in mg
	ProductName        string   `json:"productName"`
	Confidence         float64  `json:"confidence"` // 0-1
	Explanation        string   `json:"explanation"`
	BasicCalories      float64  `json:"basicCalories"` // per 100g
	BasicProtein       float64  `json:"basicProtein"`
	BasicFat           float64  `json:"basicFat"`
	BasicCarbs         float64  `json:"basicCarbs"`
	BasicFiber         *float64 `json:"basicFiber"`       // per 100g, nullable
	BasicCholesterol   *float64 `json:"basicCholesterol"` // per 100g in mg, nullable
	EstimatedWeight    *float64 `json:"estimatedWeight,omitempty"` // AI-estimated weight if not provided
	WeightUnit         *string  `json:"weightUnit,omitempty"`      // "grams" or "milliliters"
	UserProvidedWeight bool     `json:"userProvidedWeight"`        // true if user provided weight
}

// LimitCheckResult represents the result of limit checking
type LimitCheckResult struct {
	Allowed      bool   `json:"allowed"`
	CurrentUsage int    `json:"currentUsage"`
	MaxLimit     int    `json:"maxLimit"`
	ResetAt      string `json:"resetAt"` // Next reset date
	Message      string `json:"message,omitempty"`
}
