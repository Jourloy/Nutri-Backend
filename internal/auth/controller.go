package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"os"

	"github.com/charmbracelet/log"
	"github.com/go-chi/chi/v5"
)

var (
	logger = log.NewWithOptions(os.Stderr, log.Options{
		Prefix: "[auth]",
		Level:  log.DebugLevel,
	})
)

type Controller struct {
	service Service
}

func NewController() *Controller {
	service := NewService(NewRepository())

	return &Controller{service: service}
}

func (c *Controller) RegisterRoutes(router chi.Router) {
	router.Route("/auth", func(r chi.Router) {
		r.Post("/register", c.Register)
		r.Post("/login", c.Login)
		r.Post("/refresh", c.Refresh)
		r.Post("/me", c.Me)
		r.Post("/view/updates", c.IncreaseViewUpdates)
		r.Delete("/me", c.DeleteMe)
	})

	logger.Info("╔═════ Auth")
	logger.Info("║   POST /register")
	logger.Info("║   POST /login")
	logger.Info("║   POST /refresh")
	logger.Info("║   POST /me")
	logger.Info("║   POST /view/updates")
	logger.Info("║ DELETE /me")
	logger.Info("╚═════")
}

func (c *Controller) setAuthCookies(w http.ResponseWriter, access, refresh string) {
	secure := true
	http.SetCookie(w, &http.Cookie{
		Name:     "jwt",
		Value:    access,
		Path:     "/",
		SameSite: http.SameSiteNoneMode,
		HttpOnly: true,
		Secure:   secure,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refresh,
		Path:     "/auth/refresh",
		SameSite: http.SameSiteNoneMode,
		HttpOnly: true,
		Secure:   secure,
	})
}

func (c *Controller) Register(w http.ResponseWriter, r *http.Request) {
	var u RegisterData
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := c.service.Register(u)
	if err != nil {
		logger.Error("Error creating user", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	c.setAuthCookies(w, resp.AccessToken, resp.RefreshToken)
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

func (c *Controller) Login(w http.ResponseWriter, r *http.Request) {
	var u LoginData
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := c.service.Login(u)
	if err != nil {
		logger.Error("Error login", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	c.setAuthCookies(w, resp.AccessToken, resp.RefreshToken)
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (c *Controller) Refresh(w http.ResponseWriter, r *http.Request) {
	// refresh-cookie хранится только на пути /auth/refresh
	rc, err := r.Cookie("refresh_token")
	if err != nil || rc.Value == "" {
		http.Error(w, "missing refresh token", http.StatusUnauthorized)
		return
	}

	resp, err := c.service.Refresh(rc.Value)
	if err != nil {
		logger.Warn("refresh failed", "err", err)
		http.Error(w, "invalid or expired refresh token", http.StatusUnauthorized)
		return
	}

	// переустанавливаем обе куки (ротация refresh — хорошая практика)
	c.setAuthCookies(w, resp.AccessToken, resp.RefreshToken)

	// можно ничего не возвращать, но удобно вернуть пользователя и новые токены
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (c *Controller) IncreaseViewUpdates(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	resp, err := c.service.IncreaseViewUpdates(context.Background(), u.Id)
	if err != nil {
		logger.Warn("Increase view update failed", "err", err)
		http.Error(w, "increase view update failed", http.StatusUnauthorized)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (c *Controller) Me(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Можно вернуть публичные поля пользователя
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(u)
}

func (c *Controller) DeleteMe(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	err := c.service.Delete(u.Id)
	if err != nil {
		logger.Warn("delete failed", "err", err)
		http.Error(w, "invalid or expired refresh token", http.StatusUnauthorized)
		return
	}

	c.setAuthCookies(w, "", "")
	w.WriteHeader(http.StatusOK)
}
