package onef

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"hrprogress/internal/auth"
	"hrprogress/internal/httpx"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Mount(r chi.Router, jwt *auth.JWTIssuer) {
	r.Route("/onef", func(r chi.Router) {
		r.Use(auth.RequireAuth(jwt))
		r.Use(requireAdmin)
		r.Post("/sync", h.sync)
		r.Get("/runs", h.runs)
		r.Get("/status", h.status)
	})
}

func requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, ok := auth.PrincipalFrom(r.Context())
		if !ok || !p.HasRole("HR_ADMIN") {
			httpx.WriteError(w, http.StatusForbidden, "FORBIDDEN", "HR_ADMIN required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) sync(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.PrincipalFrom(r.Context())
	uid := p.UserID
	res, err := h.svc.RunSync(r.Context(), TriggerManual, &uid)
	if err != nil {
		httpx.WriteError(w, http.StatusBadGateway, "ONEF_SYNC_FAILED", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

func (h *Handler) runs(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	out, err := h.svc.ListRuns(r.Context(), limit)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	if out == nil {
		out = []SyncRun{}
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) status(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"configured": h.svc.Configured(),
	})
}
