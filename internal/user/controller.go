package user

import (
	"os"

	"github.com/charmbracelet/log"
	"github.com/go-chi/chi/v5"
)

var (
	logger = log.NewWithOptions(os.Stderr, log.Options{
		Prefix: "[user]",
		Level:  log.DebugLevel,
	})
)

type Controller struct {
	service Service
}

func NewController() *Controller {
	service := NewService()

	return &Controller{service: service}
}

func (c *Controller) RegisterRoutes(router chi.Router) {
	router.Route("/user", func(r chi.Router) {
	})

	logger.Info("╔═════ User")
	logger.Info("╚═════")
}
