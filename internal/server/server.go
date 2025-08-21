package server

import (
	"net/http"
	"os"
	"time"

	"github.com/charmbracelet/log"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/jourloy/nutri-backend/internal/auth"
	"github.com/jourloy/nutri-backend/internal/database"
	"github.com/jourloy/nutri-backend/internal/fit"
	"github.com/jourloy/nutri-backend/internal/middlewares"
	"github.com/jourloy/nutri-backend/internal/product"
	"github.com/jourloy/nutri-backend/internal/template"
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

	// Middlewares
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://nutri.jourloy.com", "http://127.0.0.1"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowedHeaders:   []string{"Origin", "Content-Length", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))
	r.Use(middlewares.Logger)
	r.Use(middlewares.Auth)
	r.Use(middleware.Recoverer)

	// Handlers
	user.NewController().RegisterRoutes(r)
	auth.NewController().RegisterRoutes(r)
	fit.NewController().RegisterRoutes(r)
	product.NewController().RegisterRoutes(r)
	template.NewController().RegisterRoutes(r)
	logger.Debug("Handlers initialized", "latency", time.Since(tempTime))

	// Start server
	logger.Info("Server started", "port", 3001, "latency (total)", time.Since(totalTime))
	err := http.ListenAndServe("0.0.0.0:3001", r)
	if err != nil {
		return err
	}

	return nil
}
