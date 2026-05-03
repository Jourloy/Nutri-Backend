package responses

import (
	"encoding/json"
	"net/http"

	"github.com/jourloy/nutri02/pkg/errors"
	"github.com/jourloy/nutri02/pkg/logger"
)

// Response represents a standard API response
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorData  `json:"error,omitempty"`
}

// ErrorData represents error information in response
type ErrorData struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Success sends a successful JSON response
func Success(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := Response{
		Success: true,
		Data:    data,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Error("Failed to encode response", "error", err)
	}
}

// Created sends a 201 Created response
func Created(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	response := Response{
		Success: true,
		Data:    data,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Error("Failed to encode response", "error", err)
	}
}

// NoContent sends a 204 No Content response
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// Error sends an error JSON response
func Error(w http.ResponseWriter, appErr *errors.AppError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(appErr.Code)

	response := Response{
		Success: false,
		Error: &ErrorData{
			Code:    appErr.Code,
			Message: appErr.Message,
		},
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Error("Failed to encode error response", "error", err)
	}
}

// BadRequest sends a 400 Bad Request response
func BadRequest(w http.ResponseWriter, message string) {
	Error(w, errors.NewBadRequest(message))
}

// Unauthorized sends a 401 Unauthorized response
func Unauthorized(w http.ResponseWriter, message string) {
	Error(w, errors.NewUnauthorized(message))
}

// NotFound sends a 404 Not Found response
func NotFound(w http.ResponseWriter, message string) {
	Error(w, errors.NewNotFound(message))
}

// InternalError sends a 500 Internal Server Error response
func InternalError(w http.ResponseWriter, message string) {
	Error(w, errors.NewInternalError(message))
}
