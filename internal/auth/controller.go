package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"

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
		r.Post("/logout", c.Logout)
		r.Post("/me", c.Me)
		r.Post("/view/updates", c.IncreaseViewUpdates)
		r.Patch("/locale", c.UpdateLocale)
		r.Delete("/me", c.DeleteMe)
		r.Get("/check-username/{username}", c.CheckUsername)
	})

	logger.Info("╔═════ Auth")
	logger.Info("║   POST /register")
	logger.Info("║   POST /login")
	logger.Info("║   POST /refresh")
	logger.Info("║   POST /logout")
	logger.Info("║   POST /me")
	logger.Info("║   POST /view/updates")
	logger.Info("║   PATCH /locale")
	logger.Info("║ DELETE /me")
	logger.Info("║    GET /check-username/{username}")
	logger.Info("╚═════")
}

type updateLocaleRequest struct {
	Locale string `json:"locale"`
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
		Path:     "/api/v1/auth/refresh",
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

func (c *Controller) Logout(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := c.service.Logout(u.Id); err != nil {
		logger.Warn("logout failed", "err", err)
		http.Error(w, "failed to logout", http.StatusInternalServerError)
		return
	}

	c.setAuthCookies(w, "", "")
	w.WriteHeader(http.StatusOK)
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
		http.Error(w, "failed to delete user", http.StatusInternalServerError)
		return
	}

	c.setAuthCookies(w, "", "")
	w.WriteHeader(http.StatusOK)
}

func (c *Controller) UpdateLocale(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req updateLocaleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	locale := strings.ToLower(req.Locale)
	if locale != "ru" && locale != "en" {
		http.Error(w, "unsupported locale", http.StatusBadRequest)
		return
	}

	updated, err := c.service.UpdateLocale(r.Context(), u.Id, locale)
	if err != nil {
		logger.Error("update locale failed", "err", err)
		http.Error(w, "failed to update locale", http.StatusInternalServerError)
		return
	}
	if updated == nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(updated)
}

func (c *Controller) CheckUsername(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	if username == "" {
		http.Error(w, "username is required", http.StatusBadRequest)
		return
	}

	available, err := c.service.CheckUsernameAvailability(username)
	if err != nil {
		logger.Error("Error checking username", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]bool{"available": available})
}
