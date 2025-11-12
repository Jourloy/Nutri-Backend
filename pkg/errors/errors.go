package errors

import (
	"fmt"
	"net/http"
)

// AppError represents an application error with HTTP status code
type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// New creates a new AppError
func New(code int, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// Wrap wraps an existing error with additional context
func Wrap(code int, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// Common errors
var (
	ErrBadRequest          = New(http.StatusBadRequest, "Bad request")
	ErrUnauthorized        = New(http.StatusUnauthorized, "Unauthorized")
	ErrForbidden           = New(http.StatusForbidden, "Forbidden")
	ErrNotFound            = New(http.StatusNotFound, "Resource not found")
	ErrConflict            = New(http.StatusConflict, "Conflict")
	ErrInternalServer      = New(http.StatusInternalServerError, "Internal server error")
	ErrInvalidCredentials  = New(http.StatusUnauthorized, "Invalid credentials")
	ErrInvalidToken        = New(http.StatusUnauthorized, "Invalid token")
	ErrTokenExpired        = New(http.StatusUnauthorized, "Token expired")
	ErrValidationFailed    = New(http.StatusBadRequest, "Validation failed")
	ErrDatabaseError       = New(http.StatusInternalServerError, "Database error")
	ErrSubscriptionExpired = New(http.StatusForbidden, "Subscription expired")
)

// NewBadRequest creates a new bad request error with custom message
func NewBadRequest(message string) *AppError {
	return New(http.StatusBadRequest, message)
}

// NewUnauthorized creates a new unauthorized error with custom message
func NewUnauthorized(message string) *AppError {
	return New(http.StatusUnauthorized, message)
}

// NewNotFound creates a new not found error with custom message
func NewNotFound(message string) *AppError {
	return New(http.StatusNotFound, message)
}

// NewInternalError creates a new internal server error with custom message
func NewInternalError(message string) *AppError {
	return New(http.StatusInternalServerError, message)
}
