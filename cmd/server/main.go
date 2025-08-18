package main

import (
	"os"

	"github.com/charmbracelet/log"
	"github.com/joho/godotenv"

	"github.com/jourloy/nutri-backend/internal/lib"
	"github.com/jourloy/nutri-backend/internal/server"
)

var logger *log.Logger

// initLogger initializes logger
func initLogger() {
	logger = log.NewWithOptions(os.Stderr, log.Options{
		Prefix: `[main]`,
		Level:  log.DebugLevel,
	})

	logger.Info("Version 1.0.0")
}

func main() {
	// Initialize logger
	initLogger()

	// Load .env
	if err := godotenv.Load(); err != nil {
		logger.Fatalf("Error loading .env file: %v", err)
	}

	logger.Debug("ENV loaded")

	// Parse env
	if err := lib.ParseENV(); err != nil {
		logger.Fatalf("Error parsing env: %v", err)
	}

	logger.Debug("ENV parsed")

	// Start internal service
	if err := server.Start(); err != nil {
		logger.Fatalf("Error starting internal service: %v", err)
	}
}
