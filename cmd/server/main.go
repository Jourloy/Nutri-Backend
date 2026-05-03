package main

import (
	"github.com/joho/godotenv"

	"github.com/jourloy/nutri02/internal/lib"
	"github.com/jourloy/nutri02/internal/server"
	"github.com/jourloy/nutri02/pkg/logger"
)

func main() {
	// Initialize logger
	logger.Init()
	logger.Info("Nutri02 Backend", "version", "1.0.0")

	// Load .env
	if err := godotenv.Load(); err != nil {
		logger.Error("Error loading .env file", "error", err)
	}

	// Parse env
	if err := lib.ParseENV(); err != nil {
		logger.Fatal("Error parsing env", "error", err)
	}

	logger.Debug("ENV parsed")

	// Start internal service
	if err := server.Start(); err != nil {
		logger.Fatal("Error starting internal service", "error", err)
	}
}
