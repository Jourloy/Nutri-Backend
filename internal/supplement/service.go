package supplement

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"
)

type Service interface {
	// Templates
	GetAllTemplates(ctx context.Context) ([]SupplementTemplate, error)

	// Supplements CRUD
	CreateSupplement(ctx context.Context, userID string, req SupplementCreateRequest) (*Supplement, error)
	GetUserSupplements(ctx context.Context, userID string, activeOnly bool) ([]Supplement, error)
	GetSupplementByID(ctx context.Context, supplementID string, userID string) (*Supplement, error)
	UpdateSupplement(ctx context.Context, supplementID string, userID string, req SupplementCreateRequest) (*Supplement, error)
	DeleteSupplement(ctx context.Context, supplementID string, userID string) error

	// Today's intakes
	CalculateTodayIntakes(ctx context.Context, userID string, timezone string) ([]TodaySupplementIntake, error)

	// Intakes
	CreateIntake(ctx context.Context, req IntakeCreateRequest, userID string, timezone string) (*SupplementIntake, error)
	DeleteIntake(ctx context.Context, intakeID string, userID string) error
	GetIntakeHistory(ctx context.Context, userID string, params IntakeHistoryParams) ([]SupplementIntake, error)

	// Statistics
	GetStatistics(ctx context.Context, userID string) (*SupplementStatistics, error)

	// Missed intakes
	GetMissedIntakesFromYesterday(ctx context.Context, userID string, timezone string) ([]TodaySupplementIntake, error)

	// Helper methods
	ShouldTakeToday(schedule SupplementSchedule, supplement Supplement, now time.Time) bool
	GetNotificationTime(schedule SupplementSchedule) string
}

type service struct {
	repo Repository
}

func NewService() Service {
	return &service{repo: NewRepository()}
}

// === Templates ===

func (s *service) GetAllTemplates(ctx context.Context) ([]SupplementTemplate, error) {
	return s.repo.GetAllTemplates(ctx)
}

// === Supplements CRUD ===

func (s *service) CreateSupplement(ctx context.Context, userID string, req SupplementCreateRequest) (*Supplement, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Create supplement
	supplement, err := s.repo.CreateSupplement(ctx, userID, req)
	if err != nil {
		return nil, err
	}

	// Create schedules
	schedules, err := s.repo.CreateSchedules(ctx, supplement.ID, req.Schedules)
	if err != nil {
		return nil, err
	}

	supplement.Schedules = schedules

	// Load template if exists
	if supplement.TemplateID != nil {
		template, err := s.repo.GetTemplateByID(ctx, *supplement.TemplateID)
		if err == nil && template != nil {
			supplement.Template = template
		}
	}

	return supplement, nil
}

func (s *service) GetUserSupplements(ctx context.Context, userID string, activeOnly bool) ([]Supplement, error) {
	supplements, err := s.repo.GetUserSupplements(ctx, userID, activeOnly)
	if err != nil {
		return nil, err
	}

	// Load schedules for each supplement
	for i := range supplements {
		schedules, err := s.repo.GetSchedulesBySupplementID(ctx, supplements[i].ID)
		if err != nil {
			return nil, err
		}
		supplements[i].Schedules = schedules
	}

	return supplements, nil
}

func (s *service) GetSupplementByID(ctx context.Context, supplementID string, userID string) (*Supplement, error) {
	supplement, err := s.repo.GetSupplementByID(ctx, supplementID, userID)
	if err != nil {
		return nil, err
	}

	// Load schedules
	schedules, err := s.repo.GetSchedulesBySupplementID(ctx, supplement.ID)
	if err != nil {
		return nil, err
	}
	supplement.Schedules = schedules

	return supplement, nil
}

func (s *service) UpdateSupplement(ctx context.Context, supplementID string, userID string, req SupplementCreateRequest) (*Supplement, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Update supplement
	supplement, err := s.repo.UpdateSupplement(ctx, supplementID, userID, req)
	if err != nil {
		return nil, err
	}

	// Delete existing schedules
	if err := s.repo.DeleteSchedulesBySupplementID(ctx, supplementID); err != nil {
		return nil, err
	}

	// Create new schedules
	schedules, err := s.repo.CreateSchedules(ctx, supplement.ID, req.Schedules)
	if err != nil {
		return nil, err
	}

	supplement.Schedules = schedules

	// Load template if exists
	if supplement.TemplateID != nil {
		template, err := s.repo.GetTemplateByID(ctx, *supplement.TemplateID)
		if err == nil && template != nil {
			supplement.Template = template
		}
	}

	return supplement, nil
}

func (s *service) DeleteSupplement(ctx context.Context, supplementID string, userID string) error {
	return s.repo.DeleteSupplement(ctx, supplementID, userID)
}

// === Today's Intakes ===

func (s *service) CalculateTodayIntakes(ctx context.Context, userID string, timezone string) ([]TodaySupplementIntake, error) {
	// Get user's active supplements with schedules
	supplements, err := s.repo.GetActiveSupplementsWithSchedules(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Load timezone
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}

	now := time.Now().In(loc)
	today := now.Format("2006-01-02")

	// Get today as date-only (midnight) for comparison
	todayMidnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	var result []TodaySupplementIntake

	for _, supplement := range supplements {
		// Compare dates only (not time) - convert to midnight in user's timezone
		startDateMidnight := time.Date(supplement.StartDate.Year(), supplement.StartDate.Month(), supplement.StartDate.Day(), 0, 0, 0, 0, loc)

		// Skip if start_date is in future
		if startDateMidnight.After(todayMidnight) {
			continue
		}

		// Skip if end_date is in past
		if supplement.EndDate != nil {
			endDateMidnight := time.Date(supplement.EndDate.Year(), supplement.EndDate.Month(), supplement.EndDate.Day(), 0, 0, 0, 0, loc)
			if endDateMidnight.Before(todayMidnight) {
				continue
			}
		}

		for _, schedule := range supplement.Schedules {
			shouldTakeToday := s.ShouldTakeToday(schedule, supplement, now)

			if shouldTakeToday {
				intakeTime := s.GetNotificationTime(schedule)

				// Check if already taken today
				intake, _ := s.repo.GetIntakeForToday(ctx, supplement.ID, schedule.ID, today)

				// Check if missed yesterday - only if supplement was active yesterday
				yesterdayTime := now.AddDate(0, 0, -1)
				yesterdayStr := yesterdayTime.Format("2006-01-02")
				missedYesterday := false

				// Only check for missed if supplement started before or on yesterday
				startDateOnly := time.Date(supplement.StartDate.Year(), supplement.StartDate.Month(), supplement.StartDate.Day(), 0, 0, 0, 0, supplement.StartDate.Location())
				yesterdayOnly := time.Date(yesterdayTime.Year(), yesterdayTime.Month(), yesterdayTime.Day(), 0, 0, 0, 0, yesterdayTime.Location())

				if !startDateOnly.After(yesterdayOnly) && s.ShouldTakeToday(schedule, supplement, yesterdayTime) {
					yesterdayIntake, _ := s.repo.GetIntakeForToday(ctx, supplement.ID, schedule.ID, yesterdayStr)
					if yesterdayIntake == nil {
						missedYesterday = true
					} else if yesterdayIntake.IsMissed {
						missedYesterday = true
					}
				}

				// Get supplement name
				supplementName := supplement.GetSupplementName("en")

				// Get icon
				var icon *string
				if supplement.Template != nil {
					icon = supplement.Template.Icon
				}

				result = append(result, TodaySupplementIntake{
					SupplementID:      supplement.ID,
					SupplementName:    supplementName,
					ScheduleID:        schedule.ID,
					IntakeTime:        intakeTime,
					IsTaken:           intake != nil && !intake.IsMissed,
					IntakeID:          func() *string { if intake != nil { return &intake.ID }; return nil }(),
					TakenAt:           func() *time.Time { if intake != nil { return &intake.TakenAt }; return nil }(),
					IsMissed:          intake != nil && intake.IsMissed,
					IsMissedYesterday: missedYesterday,
					Icon:              icon,
				})
			}
		}
	}

	return result, nil
}

// ShouldTakeToday determines if a supplement should be taken today based on schedule
func (s *service) ShouldTakeToday(schedule SupplementSchedule, supplement Supplement, now time.Time) bool {
	weekday := int(now.Weekday()) // 0=Sunday, 6=Saturday

	switch schedule.FrequencyType {
	case "times_per_day", "once_per_day":
		// Check if today's weekday is in days_of_week
		return schedule.DaysOfWeek.Contains(weekday)

	case "every_n_days":
		// Calculate days since start_date (comparing date-only, not time)
		// Convert both dates to midnight in the same location for proper comparison
		startDateMidnight := time.Date(supplement.StartDate.Year(), supplement.StartDate.Month(), supplement.StartDate.Day(), 0, 0, 0, 0, now.Location())
		nowMidnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		daysSinceStart := int(nowMidnight.Sub(startDateMidnight).Hours() / 24)
		return daysSinceStart%*schedule.IntervalDays == 0

	case "once_per_week":
		// Check if today is the specified day
		if len(schedule.DaysOfWeek) > 0 {
			return int(schedule.DaysOfWeek[0]) == weekday
		}
		return false

	case "once_per_month":
		// Check if today is the specified day of month
		return now.Day() == *schedule.DayOfMonth
	}

	return false
}

// GetNotificationTime returns the time string for a schedule
func (s *service) GetNotificationTime(schedule SupplementSchedule) string {
	if schedule.IntakeTime != nil {
		return *schedule.IntakeTime
	}
	if schedule.NotificationTime != nil {
		return *schedule.NotificationTime
	}
	return "07:00" // Default
}

// === Intakes ===

func (s *service) CreateIntake(ctx context.Context, req IntakeCreateRequest, userID string, timezone string) (*SupplementIntake, error) {
	// Get supplement to ensure it exists and belongs to user
	supplement, err := s.repo.GetSupplementByID(ctx, req.SupplementID, userID)
	if err != nil {
		return nil, fmt.Errorf("supplement not found: %w", err)
	}

	// Load timezone
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}

	now := time.Now().In(loc)
	takenAt := req.TakenAt.In(loc)

	// Determine scheduled time
	scheduledAt := takenAt
	if req.ScheduleID != nil {
		// Find the schedule
		schedules, _ := s.repo.GetSchedulesBySupplementID(ctx, req.SupplementID)
		for _, schedule := range schedules {
			if schedule.ID == *req.ScheduleID {
				// Parse intake time
				if schedule.IntakeTime != nil {
					intakeTime := *schedule.IntakeTime
					// Parse HH:MM
					var hour, minute int
					fmt.Sscanf(intakeTime, "%d:%d", &hour, &minute)
					scheduledAt = time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, loc)
				}
				break
			}
		}
	}

	// Check if taken on time (within 1 hour)
	isOnTime := math.Abs(takenAt.Sub(scheduledAt).Hours()) <= 1

	// Create intake
	intake := SupplementIntake{
		SupplementID: supplement.ID,
		ScheduleID:   req.ScheduleID,
		UserID:       userID,
		ScheduledAt:  scheduledAt,
		TakenAt:      takenAt,
		IsOnTime:     isOnTime,
		IsMissed:     false,
		Source:       req.Source,
	}

	return s.repo.CreateIntake(ctx, intake)
}

func (s *service) DeleteIntake(ctx context.Context, intakeID string, userID string) error {
	// Verify intake belongs to user before deleting
	intake, err := s.repo.GetIntakeByID(ctx, intakeID)
	if err != nil {
		return fmt.Errorf("intake not found: %w", err)
	}

	if intake.UserID != userID {
		return errors.New("unauthorized: intake does not belong to user")
	}

	return s.repo.DeleteIntake(ctx, intakeID)
}

func (s *service) GetIntakeHistory(ctx context.Context, userID string, params IntakeHistoryParams) ([]SupplementIntake, error) {
	return s.repo.GetIntakeHistory(ctx, userID, params)
}

// === Statistics ===

func (s *service) GetStatistics(ctx context.Context, userID string) (*SupplementStatistics, error) {
	return s.repo.GetStatistics(ctx, userID)
}

// === Missed Intakes ===

func (s *service) GetMissedIntakesFromYesterday(ctx context.Context, userID string, timezone string) ([]TodaySupplementIntake, error) {
	// Load timezone
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}

	yesterday := time.Now().In(loc).AddDate(0, 0, -1)
	yesterdayStr := yesterday.Format("2006-01-02")

	// Get yesterday as date-only (midnight) for comparison
	yesterdayMidnight := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, loc)

	// Get user's active supplements with schedules
	supplements, err := s.repo.GetActiveSupplementsWithSchedules(ctx, userID)
	if err != nil {
		return nil, err
	}

	var result []TodaySupplementIntake

	for _, supplement := range supplements {
		// Compare dates only (not time) - convert to midnight in user's timezone
		startDateMidnight := time.Date(supplement.StartDate.Year(), supplement.StartDate.Month(), supplement.StartDate.Day(), 0, 0, 0, 0, loc)

		// Skip if start_date is after yesterday
		if startDateMidnight.After(yesterdayMidnight) {
			continue
		}

		// Skip if end_date is before yesterday
		if supplement.EndDate != nil {
			endDateMidnight := time.Date(supplement.EndDate.Year(), supplement.EndDate.Month(), supplement.EndDate.Day(), 0, 0, 0, 0, loc)
			if endDateMidnight.Before(yesterdayMidnight) {
				continue
			}
		}

		for _, schedule := range supplement.Schedules {
			shouldTakeYesterday := s.ShouldTakeToday(schedule, supplement, yesterday)

			if shouldTakeYesterday {
				// Check if taken yesterday
				intake, _ := s.repo.GetIntakeForToday(ctx, supplement.ID, schedule.ID, yesterdayStr)

				// If not taken or marked as missed
				if intake == nil || intake.IsMissed {
					intakeTime := s.GetNotificationTime(schedule)
					supplementName := supplement.GetSupplementName("en")

					var icon *string
					if supplement.Template != nil {
						icon = supplement.Template.Icon
					}

					result = append(result, TodaySupplementIntake{
						SupplementID:   supplement.ID,
						SupplementName: supplementName,
						ScheduleID:     schedule.ID,
						IntakeTime:     intakeTime,
						IsTaken:        false,
						IsMissed:       true,
						Icon:           icon,
					})
				}
			}
		}
	}

	return result, nil
}

// === Helper Functions ===

// ValidateScheduleForFrequency validates a schedule based on frequency type
func ValidateScheduleForFrequency(schedule SupplementScheduleCreate) error {
	switch schedule.FrequencyType {
	case "times_per_day", "once_per_day":
		if schedule.IntakeTime == nil {
			return errors.New("intake_time is required for times_per_day/once_per_day")
		}
	case "every_n_days":
		if schedule.IntervalDays == nil || *schedule.IntervalDays < 1 {
			return errors.New("interval_days must be >= 1")
		}
	case "once_per_week":
		if len(schedule.DaysOfWeek) != 1 {
			return errors.New("exactly one day of week required for once_per_week")
		}
	case "once_per_month":
		if schedule.DayOfMonth == nil || *schedule.DayOfMonth < 1 || *schedule.DayOfMonth > 31 {
			return errors.New("day_of_month must be between 1 and 31")
		}
	default:
		return errors.New("invalid frequency_type")
	}
	return nil
}
