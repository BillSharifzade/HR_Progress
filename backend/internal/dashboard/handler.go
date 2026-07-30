package dashboard

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"hrprogress/internal/auth"
	"hrprogress/internal/httpx"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Mount(r chi.Router, jwt *auth.JWTIssuer) {
	r.Route("/dashboard", func(r chi.Router) {
		r.Use(auth.RequireAuth(jwt))
		r.Get("/overview", h.overview)
	})
}

// analyticsRoles may read org-wide analytics. Individual marks are never in
// this payload — everything is an aggregate — but headcount and competency
// averages are still leadership information, so a rank-and-file worker gets a
// 403 and the page falls back to their personal panel.
var analyticsRoles = []string{"HR_ADMIN", "ATS", "DEPT_HEAD", "SECTION_HEAD"}

func (h *Handler) overview(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.PrincipalFrom(r.Context())
	allowed := false
	for _, role := range analyticsRoles {
		if p.HasRole(role) {
			allowed = true
			break
		}
	}
	if !allowed {
		httpx.WriteError(w, http.StatusForbidden, "FORBIDDEN", "аналитика доступна руководителям и HR")
		return
	}
	out, err := h.svc.Overview(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}
