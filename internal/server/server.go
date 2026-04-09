package server

import (
	"net/http"
	"os"
	"time"

	"github.com/charmbracelet/log"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/jourloy/somivyn/internal/achievement"
	"github.com/jourloy/somivyn/internal/ad"
	"github.com/jourloy/somivyn/internal/admin"
	"github.com/jourloy/somivyn/internal/ai"
	"github.com/jourloy/somivyn/internal/analytics"
	"github.com/jourloy/somivyn/internal/auth"
	"github.com/jourloy/somivyn/internal/blog"
	"github.com/jourloy/somivyn/internal/body"
	"github.com/jourloy/somivyn/internal/cache"
	"github.com/jourloy/somivyn/internal/consent"
	"github.com/jourloy/somivyn/internal/database"
	"github.com/jourloy/somivyn/internal/feature"
	"github.com/jourloy/somivyn/internal/feedback"
	"github.com/jourloy/somivyn/internal/fit"
	"github.com/jourloy/somivyn/internal/internal_api"
	"github.com/jourloy/somivyn/internal/middlewares"
	"github.com/jourloy/somivyn/internal/news"
	"github.com/jourloy/somivyn/internal/order"
	"github.com/jourloy/somivyn/internal/plan"
	"github.com/jourloy/somivyn/internal/product"
	"github.com/jourloy/somivyn/internal/promo"
	"github.com/jourloy/somivyn/internal/recipe"
	"github.com/jourloy/somivyn/internal/recommendation"
	"github.com/jourloy/somivyn/internal/subscription"
	"github.com/jourloy/somivyn/internal/supplement"
	"github.com/jourloy/somivyn/internal/telegram"
	"github.com/jourloy/somivyn/internal/template"
	"github.com/jourloy/somivyn/internal/ticket"
	"github.com/jourloy/somivyn/internal/translation"
	"github.com/jourloy/somivyn/internal/user"
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
		AllowedOrigins:   []string{"https://somivyn.com", "https://nutri.jourloy.com", "http://127.0.0.1", "http://192.168.31.138"},
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
		consent.NewController().RegisterRoutes(r)
		promo.NewController().RegisterRoutes(r)
		ticket.NewController().RegisterRoutes(r)
		recommendation.NewController().RegisterRoutes(r)
		supplement.NewController().RegisterRoutes(r)

		if newsCtrl, err := news.NewController(); err != nil {
			logger.Warn("News controller disabled", "error", err)
		} else {
			newsCtrl.RegisterRoutes(r)
		}

		if aiCtrl, err := ai.NewController(); err != nil {
			logger.Warn("AI controller disabled", "error", err)
		} else {
			aiCtrl.RegisterRoutes(r)
		}

		if blogCtrl, err := blog.NewController(); err != nil {
			logger.Warn("Blog controller disabled", "error", err)
		} else {
			blogCtrl.RegisterRoutes(r)
		}

		if recipeCtrl, err := recipe.NewController(); err != nil {
			logger.Warn("Recipe controller disabled", "error", err)
		} else {
			recipeCtrl.RegisterRoutes(r)
		}

		// Admin routes (requires admin middleware)
		r.Group(func(r chi.Router) {
			r.Use(middlewares.AdminOnly)
			admin.NewController().RegisterRoutes(r)
			promo.NewController().RegisterAdminRoutes(r)
			ticket.NewController().RegisterAdminRoutes(r)
		})

		// Internal API routes (requires service token)
		r.Group(func(r chi.Router) {
			r.Use(middlewares.ServiceAuth)
			r.Route("/internal", func(r chi.Router) {
				// Supplement internal routes
				supplement.NewController().RegisterInternalRoutes(r)

				// Other internal API routes
				if internalCtrl, err := internal_api.NewController(); err != nil {
					logger.Warn("Internal API controller disabled", "error", err)
				} else {
					// Register routes directly without /internal prefix
					r.Post("/ai/analyze-food", internalCtrl.AnalyzeFoodImage)
					r.Post("/product", internalCtrl.CreateProduct)
					r.Get("/user-by-telegram-id", internalCtrl.GetUserByTelegramId)
					r.Get("/products-today-count", internalCtrl.GetProductsTodayCount)

					logger.Info("╔═════ Internal API")
					logger.Info("║   POST /ai/analyze-food")
					logger.Info("║   POST /product")
					logger.Info("║    GET /user-by-telegram-id")
					logger.Info("║    GET /products-today-count")
					logger.Info("╚═════")
				}
			})
		})
	})

	// Background workers
	order.StartWorker()
	body.StartWorker()
	supplement.StartWorker()

	logger.Debug("Handlers initialized", "latency", time.Since(tempTime))

	// Start server
	logger.Info("Server started", "port", 3002, "latency (total)", time.Since(totalTime))
	err := http.ListenAndServe("0.0.0.0:3002", r)
	if err != nil {
		return err
	}

	return nil
}
