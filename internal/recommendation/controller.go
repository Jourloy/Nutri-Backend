package recommendation

import (
	"context"
	"encoding/json"
	"net/http"
	"os"

	"github.com/charmbracelet/log"
	"github.com/go-chi/chi/v5"
	"github.com/jourloy/somivyn/internal/auth"
	"github.com/jourloy/somivyn/pkg/timeutil"
)

var (
	logger = log.NewWithOptions(os.Stderr, log.Options{
		Prefix: "[recom]",
		Level:  log.DebugLevel,
	})
)

type Controller struct {
	service Service
}

func NewController() *Controller {
	return &Controller{service: NewService()}
}

func (c *Controller) RegisterRoutes(router chi.Router) {
	router.Route("/recommendations", func(r chi.Router) {
		r.Get("/today", c.GetDailyRecommendations)
	})

	logger.Info("╔═════ Recommendations")
	logger.Info("║    GET /today")
	logger.Info("╚═════")
}

func (c *Controller) GetDailyRecommendations(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Get today's date based on user's timezone
	today := timeutil.CurrentDateForTimezone(timeutil.GetTimezoneOrDefault(u.Timezone))

	resp, err := c.service.GetDailyRecommendations(context.Background(), u.Id, today)
	if err != nil {
		logger.Error("Error getting recommendations", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
