package validator

import (
	"regexp"
	"strings"
)

// IsValidEmail checks if the email is valid
func IsValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// IsValidPassword checks if the password meets requirements
// At least 8 characters
func IsValidPassword(password string) bool {
	return len(password) >= 8
}

// IsNotEmpty checks if a string is not empty
func IsNotEmpty(s string) bool {
	return strings.TrimSpace(s) != ""
}

// IsValidUUID checks if the string is a valid UUID
func IsValidUUID(uuid string) bool {
	uuidRegex := regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	return uuidRegex.MatchString(uuid)
}

// IsPositive checks if a number is positive
func IsPositive(n int) bool {
	return n > 0
}

// IsPositiveFloat checks if a float is positive
func IsPositiveFloat(n float64) bool {
	return n > 0
}

// IsInRange checks if a number is within a range
func IsInRange(n, min, max int) bool {
	return n >= min && n <= max
}

// IsInRangeFloat checks if a float is within a range
func IsInRangeFloat(n, min, max float64) bool {
	return n >= min && n <= max
}

// HasMinLength checks if a string has minimum length
func HasMinLength(s string, minLen int) bool {
	return len(s) >= minLen
}

// HasMaxLength checks if a string has maximum length
func HasMaxLength(s string, maxLen int) bool {
	return len(s) <= maxLen
}
