package server

import (
	"net/http"
	"os"
	"time"

	"github.com/charmbracelet/log"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/jourloy/nutri-backend/internal/achievement"
	"github.com/jourloy/nutri-backend/internal/ad"
	"github.com/jourloy/nutri-backend/internal/admin"
	"github.com/jourloy/nutri-backend/internal/ai"
	"github.com/jourloy/nutri-backend/internal/analytics"
	"github.com/jourloy/nutri-backend/internal/auth"
	"github.com/jourloy/nutri-backend/internal/body"
	"github.com/jourloy/nutri-backend/internal/cache"
	"github.com/jourloy/nutri-backend/internal/database"
	"github.com/jourloy/nutri-backend/internal/feature"
	"github.com/jourloy/nutri-backend/internal/feedback"
	"github.com/jourloy/nutri-backend/internal/fit"
	"github.com/jourloy/nutri-backend/internal/middlewares"
	"github.com/jourloy/nutri-backend/internal/order"
	"github.com/jourloy/nutri-backend/internal/plan"
	"github.com/jourloy/nutri-backend/internal/product"
	"github.com/jourloy/nutri-backend/internal/promo"
	"github.com/jourloy/nutri-backend/internal/subscription"
	"github.com/jourloy/nutri-backend/internal/telegram"
	"github.com/jourloy/nutri-backend/internal/template"
	"github.com/jourloy/nutri-backend/internal/ticket"
	"github.com/jourloy/nutri-backend/internal/translation"
	"github.com/jourloy/nutri-backend/internal/user"
)

var (
	logger = log.NewWithOptions(os.Stderr, log.Options{
		Prefix: "[srvr]",
		Level:  log.DebugLevel,
	})
)

func Start() error {
	totalTime := time.Now()
	tempTime := time.Now()

	r := chi.NewRouter()

	database.Connect()
	logger.Debug("Repositories initialized", "latency", time.Since(tempTime))
	tempTime = time.Now()
	if err := cache.Connect(); err != nil {
		logger.Warn("Redis cache disabled", "error", err)
	}

	// Middlewares
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://nutri.jourloy.com", "http://127.0.0.1"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowedHeaders:   []string{"Origin", "Content-Length", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))
	r.Use(middlewares.Logger)
	r.Use(middlewares.Auth)
	r.Use(middlewares.Subscription)
	r.Use(middleware.Recoverer)

	r.Route("/api/v1", func(r chi.Router) {
		user.NewController().RegisterRoutes(r)
		auth.NewController().RegisterRoutes(r)
		fit.NewController().RegisterRoutes(r)
		product.NewController().RegisterRoutes(r)
		plan.NewController().RegisterRoutes(r)
		order.NewController().RegisterRoutes(r)
		feature.NewController().RegisterRoutes(r)
		subscription.NewController().RegisterRoutes(r)
		template.NewController().RegisterRoutes(r)
		telegram.NewController().RegisterRoutes(r)
		achievement.NewController().RegisterRoutes(r)
		analytics.NewController().RegisterRoutes(r)
		ad.NewController().RegisterRoutes(r)
		body.NewController().RegisterRoutes(r)
		feedback.NewController().RegisterRoutes(r)
		translation.NewController().RegisterRoutes(r)
		promo.NewController().RegisterRoutes(r)
		ticket.NewController().RegisterRoutes(r)

		if aiCtrl, err := ai.NewController(); err != nil {
			logger.Warn("AI controller disabled", "error", err)
		} else {
			aiCtrl.RegisterRoutes(r)
		}

		// Admin routes (requires admin middleware)
		r.Group(func(r chi.Router) {
			r.Use(middlewares.AdminOnly)
			admin.NewController().RegisterRoutes(r)
			promo.NewController().RegisterAdminRoutes(r)
			ticket.NewController().RegisterAdminRoutes(r)
		})
	})

	// Background workers
	order.StartWorker()
	body.StartWorker()

	logger.Debug("Handlers initialized", "latency", time.Since(tempTime))

	// Start server
	logger.Info("Server started", "port", 3002, "latency (total)", time.Since(totalTime))
	err := http.ListenAndServe("0.0.0.0:3002", r)
	if err != nil {
		return err
	}

	return nil
}
