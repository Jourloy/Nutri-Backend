package main

import (
	"github.com/joho/godotenv"

	"github.com/jourloy/somivyn/internal/lib"
	"github.com/jourloy/somivyn/internal/server"
	"github.com/jourloy/somivyn/pkg/logger"
)

func main() {
	// Initialize logger
	logger.Init()
	logger.Info("Somivyn Backend", "version", "1.0.0")

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
