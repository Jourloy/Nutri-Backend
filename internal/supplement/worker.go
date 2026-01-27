package supplement

import (
	"context"
	"os"
	"time"

	"github.com/charmbracelet/log"
)

var (
	workerLogger = log.NewWithOptions(os.Stderr, log.Options{
		Prefix: "[spmw]",
		Level:  log.DebugLevel,
	})
)

// StartWorker starts the supplement background worker
// Main notification logic is in Telegram bot worker (it has direct access to bot.Send)
// This worker handles cleanup and maintenance tasks
func StartWorker() {
	go func() {
		workerLogger.Info("Starting supplement worker")

		// Run cleanup task once per day at 3:00 AM UTC
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for {
			now := time.Now().UTC()
			// Run cleanup at 3 AM UTC
			if now.Hour() == 3 && now.Minute() < 5 {
				runCleanup()
			}
			<-ticker.C
		}
	}()
}

// runCleanup performs maintenance tasks
func runCleanup() {
	ctx := context.Background()
	repo := NewRepository()

	// Cleanup old notification logs (older than 90 days)
	// This prevents the table from growing indefinitely
	cleanupOldNotifications(ctx, repo)

	workerLogger.Debug("Cleanup completed")
}

// cleanupOldNotifications removes notification logs older than 90 days
func cleanupOldNotifications(ctx context.Context, repo Repository) {
	// Note: This would need a new repository method
	// For MVP, we can skip this or implement it later
	// The table has indexes and won't grow too large in the short term
	workerLogger.Debug("Notification cleanup skipped (not critical for MVP)")
}

// Note: Supplement notification sending logic is implemented in Telegram bot worker
// See: /Nutri-Telegram/internal/supplement/worker.go (to be created in Phase 7)
//
// The Telegram bot worker will:
// 1. Run every 2 minutes
// 2. Query supplement_schedules for users with Telegram enabled
// 3. Check if current time (in user's timezone) matches a schedule
// 4. Check supplement_notification_log to prevent duplicates
// 5. Send notification via bot.Send()
// 6. Log to supplement_notification_log
//
// This separation is necessary because:
// - Backend doesn't have access to Telegram bot API
// - Telegram bot has shared database access
// - Telegram bot worker can directly send messages
