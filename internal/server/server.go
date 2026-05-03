package server

import (
	"net/http"
	"os"
	"time"

	"github.com/charmbracelet/log"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/jourloy/nutri02/internal/achievement"
	"github.com/jourloy/nutri02/internal/ad"
	"github.com/jourloy/nutri02/internal/admin"
	"github.com/jourloy/nutri02/internal/ai"
	"github.com/jourloy/nutri02/internal/analytics"
	"github.com/jourloy/nutri02/internal/auth"
	"github.com/jourloy/nutri02/internal/blog"
	"github.com/jourloy/nutri02/internal/body"
	"github.com/jourloy/nutri02/internal/cache"
	"github.com/jourloy/nutri02/internal/consent"
	"github.com/jourloy/nutri02/internal/database"
	"github.com/jourloy/nutri02/internal/feature"
	"github.com/jourloy/nutri02/internal/feedback"
	"github.com/jourloy/nutri02/internal/fit"
	"github.com/jourloy/nutri02/internal/internal_api"
	"github.com/jourloy/nutri02/internal/middlewares"
	"github.com/jourloy/nutri02/internal/news"
	"github.com/jourloy/nutri02/internal/order"
	"github.com/jourloy/nutri02/internal/plan"
	"github.com/jourloy/nutri02/internal/product"
	"github.com/jourloy/nutri02/internal/promo"
	"github.com/jourloy/nutri02/internal/recipe"
	"github.com/jourloy/nutri02/internal/recommendation"
	"github.com/jourloy/nutri02/internal/subscription"
	"github.com/jourloy/nutri02/internal/supplement"
	"github.com/jourloy/nutri02/internal/telegram"
	"github.com/jourloy/nutri02/internal/template"
	"github.com/jourloy/nutri02/internal/ticket"
	"github.com/jourloy/nutri02/internal/translation"
	"github.com/jourloy/nutri02/internal/user"
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
	r.Use(redirectLegacyHosts)
	r.Use(cors.Handler(newCORSOptions([]string{
		"https://nutri02.com",
		"https://www.nutri02.com",
		"https://somivyn.com",
		"https://www.somivyn.com",
		"https://somivyn.jourloy.com",
		"https://nutri.jourloy.com",
		"https://nutri02.jourloy.com",
		"https://api.somivyn.com",
		"https://api.somivyn.jourloy.com",
		"https://api-somivyn.jourloy.com",
		"http://192.168.31.138",
	})))
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
