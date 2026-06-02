package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"

	"hrprogress/internal/httpx"
)

const refreshCookieName = "hr_refresh"

type Handler struct {
	svc      *Service
	validate *validator.Validate
	cookieSecure bool
	refreshTTL   time.Duration
}

func NewHandler(svc *Service, refreshTTL time.Duration, cookieSecure bool) *Handler {
	return &Handler{
		svc: svc,
		validate: validator.New(validator.WithRequiredStructEnabled()),
		cookieSecure: cookieSecure,
		refreshTTL: refreshTTL,
	}
}

func (h *Handler) Mount(r chi.Router, jwt *JWTIssuer) {
	r.Route("/auth", func(r chi.Router) {
		r.Post("/login", h.login)
		r.Post("/refresh", h.refresh)
		r.Post("/logout", h.logout)
		r.Group(func(r chi.Router) {
			r.Use(RequireAuth(jwt))
			r.Get("/me", h.me)
		})
	})
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_JSON", "invalid JSON")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION", err.Error())
		return
	}
	resp, refresh, err := h.svc.Login(r.Context(), req.Username, req.Password, clientIP(r), r.UserAgent())
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			httpx.WriteError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "неверные имя пользователя или пароль")
		case errors.Is(err, ErrInactiveUser):
			httpx.WriteError(w, http.StatusForbidden, "INACTIVE_USER", "учётная запись отключена")
		default:
			httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		}
		return
	}
	h.setRefreshCookie(w, refresh)
	httpx.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(refreshCookieName)
	if err != nil || c.Value == "" {
		httpx.WriteError(w, http.StatusUnauthorized, "NO_REFRESH", "missing refresh cookie")
		return
	}
	resp, refresh, err := h.svc.Refresh(r.Context(), c.Value, clientIP(r), r.UserAgent())
	if err != nil {
		h.clearRefreshCookie(w)
		httpx.WriteError(w, http.StatusUnauthorized, "REFRESH_FAILED", "refresh failed")
		return
	}
	h.setRefreshCookie(w, refresh)
	httpx.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(refreshCookieName); err == nil && c.Value != "" {
		_ = h.svc.Logout(r.Context(), c.Value)
	}
	h.clearRefreshCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	p, _ := PrincipalFrom(r.Context())
	u, err := h.svc.Me(r.Context(), p.UserID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, u)
}

func (h *Handler) setRefreshCookie(w http.ResponseWriter, value string) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    value,
		Path:     "/api/v1/auth",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(h.refreshTTL.Seconds()),
	})
}

func (h *Handler) clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     "/api/v1/auth",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func clientIP(r *http.Request) string {
	if v := r.Header.Get("X-Forwarded-For"); v != "" {
		return v
	}
	return r.RemoteAddr
}
