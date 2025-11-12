package logger

import (
	"os"

	"github.com/charmbracelet/log"
)

var Logger *log.Logger

// Init initializes the global logger
func Init() {
	Logger = log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: true,
		TimeFormat:      "2006-01-02 15:04:05",
	})
}

// Debug logs a debug message
func Debug(msg string, keyvals ...interface{}) {
	Logger.Debug(msg, keyvals...)
}

// Info logs an info message
func Info(msg string, keyvals ...interface{}) {
	Logger.Info(msg, keyvals...)
}

// Warn logs a warning message
func Warn(msg string, keyvals ...interface{}) {
	Logger.Warn(msg, keyvals...)
}

// Error logs an error message
func Error(msg string, keyvals ...interface{}) {
	Logger.Error(msg, keyvals...)
}

// Fatal logs a fatal message and exits
func Fatal(msg string, keyvals ...interface{}) {
	Logger.Fatal(msg, keyvals...)
}

// With creates a new logger with additional key-value pairs
func With(keyvals ...interface{}) *log.Logger {
	return Logger.With(keyvals...)
}
