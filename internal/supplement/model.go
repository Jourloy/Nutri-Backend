package supplement

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
)

// IntArray is a custom type for PostgreSQL integer arrays
type IntArray pq.Int64Array

func (a IntArray) Value() (driver.Value, error) {
	return pq.Int64Array(a).Value()
}

func (a *IntArray) Scan(src interface{}) error {
	return (*pq.Int64Array)(a).Scan(src)
}

// ToIntSlice converts IntArray to []int for JSON serialization
func (a IntArray) ToIntSlice() []int {
	result := make([]int, len(a))
	for i, v := range a {
		result[i] = int(v)
	}
	return result
}

// IntSliceToIntArray converts []int to IntArray
func IntSliceToIntArray(slice []int) IntArray {
	result := make(IntArray, len(slice))
	for i, v := range slice {
		result[i] = int64(v)
	}
	return result
}

// Contains checks if an int is in the IntArray
func (a IntArray) Contains(val int) bool {
	for _, item := range a {
		if int(item) == val {
			return true
		}
	}
	return false
}

// SupplementTemplate represents a predefined supplement template
type SupplementTemplate struct {
	ID            int64     `json:"id" db:"id"`
	Slug          string    `json:"slug" db:"slug"`
	NameRu        string    `json:"nameRu" db:"name_ru"`
	NameEn        string    `json:"nameEn" db:"name_en"`
	Category      string    `json:"category" db:"category"` // 'vitamin', 'mineral', 'sports', 'other'
	Icon          *string   `json:"icon,omitempty" db:"icon"`
	DescriptionRu *string   `json:"descriptionRu,omitempty" db:"description_ru"`
	DescriptionEn *string   `json:"descriptionEn,omitempty" db:"description_en"`
	SortOrder     int       `json:"sortOrder" db:"sort_order"`
	CreatedAt     time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt     time.Time `json:"updatedAt" db:"updated_at"`
}

// Supplement represents a user's supplement
type Supplement struct {
	ID         string     `json:"id" db:"id"`
	UserID     string     `json:"userId" db:"user_id"`
	TemplateID *int64     `json:"templateId,omitempty" db:"template_id"`
	CustomName *string    `json:"customName,omitempty" db:"custom_name"`
	StartDate  time.Time  `json:"startDate" db:"start_date"`
	EndDate    *time.Time `json:"endDate,omitempty" db:"end_date"`
	IsActive   bool       `json:"isActive" db:"is_active"`
	Notes      *string    `json:"notes,omitempty" db:"notes"`
	CreatedAt  time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt  time.Time  `json:"updatedAt" db:"updated_at"`
	DeletedAt  *time.Time `json:"deletedAt,omitempty" db:"deleted_at"`

	// Joined data (not in DB)
	Template  *SupplementTemplate  `json:"template,omitempty" db:"-"`
	Schedules []SupplementSchedule `json:"schedules,omitempty" db:"-"`
}

// SupplementSchedule defines when to take a supplement
type SupplementSchedule struct {
	ID                 string    `json:"id" db:"id"`
	SupplementID       string    `json:"supplementId" db:"supplement_id"`
	FrequencyType      string    `json:"frequencyType" db:"frequency_type"`     // 'times_per_day', 'once_per_day', etc.
	IntakeTime         *string   `json:"intakeTime,omitempty" db:"intake_time"` // HH:MM format
	DaysOfWeek         IntArray  `json:"daysOfWeek,omitempty" db:"days_of_week"`
	IntervalDays       *int      `json:"intervalDays,omitempty" db:"interval_days"`
	DayOfMonth         *int      `json:"dayOfMonth,omitempty" db:"day_of_month"`
	EnableNotification bool      `json:"enableNotification" db:"enable_notification"`
	NotificationTime   *string   `json:"notificationTime,omitempty" db:"notification_time"`
	CreatedAt          time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt          time.Time `json:"updatedAt" db:"updated_at"`
}

// SupplementIntake represents an actual intake record
type SupplementIntake struct {
	ID           string    `json:"id" db:"id"`
	SupplementID string    `json:"supplementId" db:"supplement_id"`
	ScheduleID   *string   `json:"scheduleId,omitempty" db:"schedule_id"`
	UserID       string    `json:"userId" db:"user_id"`
	ScheduledAt  time.Time `json:"scheduledAt" db:"scheduled_at"`
	TakenAt      time.Time `json:"takenAt" db:"taken_at"`
	IsOnTime     bool      `json:"isOnTime" db:"is_on_time"`
	IsMissed     bool      `json:"isMissed" db:"is_missed"`
	Source       string    `json:"source" db:"source"` // 'manual', 'telegram', 'dashboard'
	CreatedAt    time.Time `json:"createdAt" db:"created_at"`

	// Joined data (not in DB)
	SupplementName string `json:"supplementName,omitempty" db:"-"`
}

// SupplementNotificationLog tracks sent notifications
type SupplementNotificationLog struct {
	ID               int64     `json:"id" db:"id"`
	SupplementID     string    `json:"supplementId" db:"supplement_id"`
	ScheduleID       string    `json:"scheduleId" db:"schedule_id"`
	UserID           string    `json:"userId" db:"user_id"`
	TelegramID       string    `json:"telegramId" db:"telegram_id"`
	ScheduledFor     time.Time `json:"scheduledFor" db:"scheduled_for"`
	SentAt           time.Time `json:"sentAt" db:"sent_at"`
	NotificationType string    `json:"notificationType" db:"notification_type"` // 'exact_time', 'morning_reminder', 'missed_reminder'
}

// === DTOs for API ===

// SupplementCreateRequest is the request body for creating a supplement
type SupplementCreateRequest struct {
	TemplateID *int64                     `json:"templateId,omitempty"`
	CustomName *string                    `json:"customName,omitempty"`
	StartDate  string                     `json:"startDate"` // YYYY-MM-DD
	EndDate    *string                    `json:"endDate,omitempty"`
	Notes      *string                    `json:"notes,omitempty"`
	Schedules  []SupplementScheduleCreate `json:"schedules"`
}

// SupplementScheduleCreate is the request body for creating a schedule
type SupplementScheduleCreate struct {
	FrequencyType      string  `json:"frequencyType"`
	IntakeTime         *string `json:"intakeTime,omitempty"`
	DaysOfWeek         []int   `json:"daysOfWeek,omitempty"`
	IntervalDays       *int    `json:"intervalDays,omitempty"`
	DayOfMonth         *int    `json:"dayOfMonth,omitempty"`
	EnableNotification bool    `json:"enableNotification"`
	NotificationTime   *string `json:"notificationTime,omitempty"`
}

// IntakeCreateRequest is the request body for marking supplement as taken
type IntakeCreateRequest struct {
	SupplementID string    `json:"supplementId"`
	ScheduleID   *string   `json:"scheduleId,omitempty"`
	TakenAt      time.Time `json:"takenAt"`
	Source       string    `json:"source"` // 'dashboard', 'telegram', 'manual'
}

// TodaySupplementIntake represents a supplement that should be taken today
type TodaySupplementIntake struct {
	SupplementID      string     `json:"supplementId"`
	SupplementName    string     `json:"supplementName"`
	ScheduleID        string     `json:"scheduleId"`
	IntakeTime        string     `json:"intakeTime"` // HH:MM
	IsTaken           bool       `json:"isTaken"`
	IntakeID          *string    `json:"intakeId,omitempty"` // ID of the intake record if taken
	TakenAt           *time.Time `json:"takenAt,omitempty"`
	IsMissed          bool       `json:"isMissed"`
	IsMissedYesterday bool       `json:"isMissedYesterday"`
	Icon              *string    `json:"icon,omitempty"`
}

// SupplementStatistics represents statistics for supplements
type SupplementStatistics struct {
	TotalSupplements  int     `json:"totalSupplements"`
	ActiveSupplements int     `json:"activeSupplements"`
	TotalIntakes      int     `json:"totalIntakes"`
	MissedIntakes     int     `json:"missedIntakes"`
	MissRate          float64 `json:"missRate"`
	CurrentStreak     int     `json:"currentStreak"`
	LongestStreak     int     `json:"longestStreak"`
}

// SupplementWithSchedules is a supplement with its schedules and template
type SupplementWithSchedules struct {
	Supplement
	Template  *SupplementTemplate  `json:"template,omitempty"`
	Schedules []SupplementSchedule `json:"schedules"`
}

// IntakeHistoryParams are query parameters for intake history
type IntakeHistoryParams struct {
	Date         *string `json:"date,omitempty"` // YYYY-MM-DD
	SupplementID *string `json:"supplementId,omitempty"`
	Limit        int     `json:"limit"`
	Offset       int     `json:"offset"`
}

// MarshalJSON for IntArray to ensure it's serialized as JSON array of ints
func (a IntArray) MarshalJSON() ([]byte, error) {
	if a == nil {
		return json.Marshal([]int{})
	}
	return json.Marshal(a.ToIntSlice())
}

// UnmarshalJSON for IntArray
func (a *IntArray) UnmarshalJSON(data []byte) error {
	var arr []int
	if err := json.Unmarshal(data, &arr); err != nil {
		return err
	}
	*a = make(IntArray, len(arr))
	for i, v := range arr {
		(*a)[i] = int64(v)
	}
	return nil
}

// Validate validates SupplementCreateRequest
func (r *SupplementCreateRequest) Validate() error {
	// Must have either template_id or custom_name
	if r.TemplateID == nil && r.CustomName == nil {
		return errors.New("either templateId or customName must be provided")
	}
	if r.TemplateID != nil && r.CustomName != nil {
		return errors.New("cannot provide both templateId and customName")
	}

	// Must have start date
	if r.StartDate == "" {
		return errors.New("startDate is required")
	}

	// Must have at least one schedule
	if len(r.Schedules) == 0 {
		return errors.New("at least one schedule is required")
	}

	// Validate each schedule
	for i, schedule := range r.Schedules {
		if err := schedule.Validate(); err != nil {
			return errors.New("schedule " + string(rune(i)) + ": " + err.Error())
		}
	}

	return nil
}

// Validate validates SupplementScheduleCreate
func (s *SupplementScheduleCreate) Validate() error {
	// Validate intake_time format if provided
	if s.IntakeTime != nil {
		if err := ValidateTimeFormat(*s.IntakeTime); err != nil {
			return fmt.Errorf("intakeTime: %w", err)
		}
	}

	// Validate notification_time format if provided
	if s.NotificationTime != nil {
		if err := ValidateTimeFormat(*s.NotificationTime); err != nil {
			return fmt.Errorf("notificationTime: %w", err)
		}
	}

	switch s.FrequencyType {
	case "times_per_day":
		if s.IntakeTime == nil {
			return errors.New("intakeTime is required for times_per_day")
		}
	case "once_per_day":
		// intakeTime is optional for once_per_day
	case "every_n_days":
		if s.IntervalDays == nil || *s.IntervalDays < 1 {
			return errors.New("intervalDays must be >= 1 for every_n_days")
		}
	case "once_per_week":
		if len(s.DaysOfWeek) != 1 {
			return errors.New("exactly one day of week is required for once_per_week")
		}
	case "once_per_month":
		if s.DayOfMonth == nil || *s.DayOfMonth < 1 || *s.DayOfMonth > 31 {
			return errors.New("dayOfMonth must be between 1 and 31 for once_per_month")
		}
	default:
		return errors.New("invalid frequencyType: " + s.FrequencyType)
	}
	return nil
}

// ValidateTimeFormat validates HH:MM format (00:00 to 23:59)
func ValidateTimeFormat(timeStr string) error {
	if len(timeStr) != 5 {
		return errors.New("time must be in HH:MM format")
	}

	var hour, minute int
	_, err := fmt.Sscanf(timeStr, "%d:%d", &hour, &minute)
	if err != nil {
		return errors.New("invalid time format, expected HH:MM")
	}

	if hour < 0 || hour > 23 {
		return errors.New("hour must be between 00 and 23")
	}

	if minute < 0 || minute > 59 {
		return errors.New("minute must be between 00 and 59")
	}

	return nil
}

// GetSupplementName returns the display name of a supplement (template or custom)
func (s *Supplement) GetSupplementName(locale string) string {
	if s.Template != nil {
		if locale == "en" {
			return s.Template.NameEn
		}
		return s.Template.NameRu
	}
	if s.CustomName != nil {
		return *s.CustomName
	}
	return "Unknown"
}
