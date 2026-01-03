package timeutil

import "time"

// CurrentDateForTimezone возвращает текущую дату в указанной timezone
// в формате "YYYY-MM-DD"
func CurrentDateForTimezone(tz string) string {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}
	return time.Now().In(loc).Format("2006-01-02")
}

// CurrentTimeForTimezone возвращает текущее время в указанной timezone
func CurrentTimeForTimezone(tz string) time.Time {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}
	return time.Now().In(loc)
}

// ValidateTimezone проверяет валидность IANA timezone
func ValidateTimezone(tz string) bool {
	_, err := time.LoadLocation(tz)
	return err == nil
}

// GetTimezoneOrDefault возвращает timezone или UTC если nil/пустой
func GetTimezoneOrDefault(tz *string) string {
	if tz == nil || *tz == "" {
		return "UTC"
	}
	// Проверяем валидность timezone
	if !ValidateTimezone(*tz) {
		return "UTC"
	}
	return *tz
}
