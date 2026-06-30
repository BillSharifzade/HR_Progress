package competency

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"hrprogress/internal/auth"
	"hrprogress/internal/httpx"
	"hrprogress/internal/rbac"
)

type Handler struct {
	svc      *Service
	validate *validator.Validate
}

func NewHandler(svc *Service) *Handler {
	return &Handler{
		svc:      svc,
		validate: validator.New(validator.WithRequiredStructEnabled()),
	}
}

func (h *Handler) Mount(r chi.Router, jwt *auth.JWTIssuer) {
	r.Route("/competencies", func(r chi.Router) {
		r.Use(auth.RequireAuth(jwt))
		r.Get("/", h.listCompetencies)
		r.Post("/", h.createCompetency)
		r.Post("/reorder", h.reorderCompetencies)
		r.Route("/{comp_id}", func(r chi.Router) {
			r.Put("/", h.updateCompetency)
			r.Delete("/", h.deleteCompetency)
		})
	})

	r.Route("/grades", func(r chi.Router) {
		r.Use(auth.RequireAuth(jwt))
		r.Get("/", h.listGrades)
	})

	r.Route("/departments", func(r chi.Router) {
		r.Use(auth.RequireAuth(jwt))
		r.Get("/", h.listDepartments)
		r.Post("/", h.createDepartment)
		r.Route("/{dept_id}", func(r chi.Router) {
			r.Put("/", h.updateDepartment)
			r.Delete("/", h.deleteDepartment)
			r.Get("/employees", h.listEmployees)
			r.Route("/requirements", func(r chi.Router) {
				r.Get("/", h.listRequirements)
				r.Put("/", h.upsertRequirements)
			})
		})
	})

	r.Route("/assessment-periods", func(r chi.Router) {
		r.Use(auth.RequireAuth(jwt))
		r.Get("/", h.listPeriods)
		r.Post("/", h.createPeriod)
		r.Route("/{period_id}", func(r chi.Router) {
			r.Get("/", h.getPeriod)
			r.Post("/scores", h.upsertScore)
			r.Post("/scores/bulk", h.bulkUpsertScores)
			r.Route("/participants", func(r chi.Router) {
				r.Get("/", h.listParticipants)
				r.With(requireAdmin).Post("/", h.addParticipants)
			})
			// Criteria (FR-AS3)
			r.Get("/criteria", h.listCriteria)
			r.With(requireAdmin).Put("/criteria", h.setCriteria)
			// Assessees (FR-AS2) + per-assessee assessors (FR-AS4)
			r.Get("/assessees", h.listAssessees)
			r.With(requireAdmin).Post("/assessees", h.addAssessees)
			r.With(requireAdmin).Delete("/assessees/{user_id}", h.removeAssessee)
			r.Get("/assessee-assessors", h.listAssesseeAssessors)
			r.With(requireAdmin).Put("/assessees/{user_id}/assessors", h.setAssesseeAssessors)
			// Lifecycle (Section 5, FR-AS9, FR-AS10)
			r.With(requireAdmin).Post("/transition", h.transitionPeriod)
			// Learning groups (FR-AS13)
			r.Get("/groups", h.listGroups)
			r.Get("/groups/journal", h.listGroupJournal)
			r.With(requireAdmin).Post("/groups/regenerate", h.regenerateGroups)
			r.With(requireAdmin).Post("/groups/move", h.moveGroupMember)
			r.With(requireAdmin).Post("/groups/confirm", h.confirmGroups)

			r.Get("/my-scores", h.listMyScores)
			r.Get("/consolidated", h.listConsolidated)
		})
	})

	// Interpretation reference / справочник (FR-AS7.2)
	r.Route("/interpretations", func(r chi.Router) {
		r.Use(auth.RequireAuth(jwt))
		r.Get("/", h.listInterpretations)
		r.Get("/lookup", h.lookupInterpretation)
		r.Get("/history", h.interpretationHistory)
		r.With(requireAdmin).Post("/", h.upsertInterpretation)
		r.With(requireAdmin).Post("/copy", h.copyInterpretations)
		r.With(requireAdmin).Delete("/{id}", h.deleteInterpretation)
	})

	r.Route("/me/assessment-periods", func(r chi.Router) {
		r.Use(auth.RequireAuth(jwt))
		r.Get("/", h.listMyPeriods)
	})

	r.Route("/me/assessment-results", func(r chi.Router) {
		r.Use(auth.RequireAuth(jwt))
		r.Get("/", h.myAssessmentResults)
	})

	r.Route("/users", func(r chi.Router) {
		r.Use(auth.RequireAuth(jwt))
		r.Get("/with-role", h.listUsersWithRole)
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

func (h *Handler) listDepartments(w http.ResponseWriter, r *http.Request) {
	var (
		depts []Department
		err   error
	)
	if r.URL.Query().Get("include_inactive") == "true" {
		depts, err = h.svc.ListAllDepartments(r.Context())
	} else {
		depts, err = h.svc.ListDepartments(r.Context())
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}
	if depts == nil {
		depts = []Department{}
	}
	httpx.WriteJSON(w, http.StatusOK, depts)
}

func (h *Handler) listGrades(w http.ResponseWriter, r *http.Request) {
	grades, err := h.svc.ListGrades(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}
	if grades == nil {
		grades = []Grade{}
	}
	httpx.WriteJSON(w, http.StatusOK, grades)
}

func (h *Handler) createDepartment(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.PrincipalFrom(r.Context())
	if !rbac.Allow(rbac.Actor{UserID: p.UserID, Roles: p.Roles}, rbac.ActionUsersCreate, rbac.Target{}) {
		httpx.WriteError(w, http.StatusForbidden, "FORBIDDEN", "insufficient permissions")
		return
	}
	var req CreateDepartmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_JSON", "invalid JSON")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION", err.Error())
		return
	}
	dept, err := h.svc.CreateDepartment(r.Context(), req)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, dept)
}

func (h *Handler) updateDepartment(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.PrincipalFrom(r.Context())
	if !rbac.Allow(rbac.Actor{UserID: p.UserID, Roles: p.Roles}, rbac.ActionUsersEdit, rbac.Target{}) {
		httpx.WriteError(w, http.StatusForbidden, "FORBIDDEN", "insufficient permissions")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "dept_id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_PARAM", "invalid dept_id")
		return
	}
	var req UpdateDepartmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_JSON", "invalid JSON")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION", err.Error())
		return
	}
	dept, err := h.svc.UpdateDepartment(r.Context(), id, req)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "department not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, dept)
}

func (h *Handler) deleteDepartment(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.PrincipalFrom(r.Context())
	if !rbac.Allow(rbac.Actor{UserID: p.UserID, Roles: p.Roles}, rbac.ActionUsersEdit, rbac.Target{}) {
		httpx.WriteError(w, http.StatusForbidden, "FORBIDDEN", "insufficient permissions")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "dept_id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_PARAM", "invalid dept_id")
		return
	}
	if err := h.svc.DeleteDepartment(r.Context(), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "department not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listCompetencies(w http.ResponseWriter, r *http.Request) {
	comps, err := h.svc.ListCompetencies(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}
	if comps == nil {
		comps = []Competency{}
	}
	httpx.WriteJSON(w, http.StatusOK, comps)
}

func (h *Handler) createCompetency(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.PrincipalFrom(r.Context())
	actor := rbac.Actor{UserID: p.UserID, Roles: p.Roles}
	if !rbac.Allow(actor, rbac.ActionUsersCreate, rbac.Target{}) {
		httpx.WriteError(w, http.StatusForbidden, "FORBIDDEN", "insufficient permissions")
		return
	}
	var req CreateCompetencyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_JSON", "invalid JSON")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION", err.Error())
		return
	}
	comp, err := h.svc.CreateCompetency(r.Context(), req)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, comp)
}

func (h *Handler) updateCompetency(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.PrincipalFrom(r.Context())
	actor := rbac.Actor{UserID: p.UserID, Roles: p.Roles}
	if !rbac.Allow(actor, rbac.ActionUsersEdit, rbac.Target{}) {
		httpx.WriteError(w, http.StatusForbidden, "FORBIDDEN", "insufficient permissions")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "comp_id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_PARAM", "invalid comp_id")
		return
	}
	var req UpdateCompetencyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_JSON", "invalid JSON")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION", err.Error())
		return
	}
	comp, err := h.svc.UpdateCompetency(r.Context(), id, req)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "competency not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, comp)
}

func (h *Handler) reorderCompetencies(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.PrincipalFrom(r.Context())
	actor := rbac.Actor{UserID: p.UserID, Roles: p.Roles}
	if !rbac.Allow(actor, rbac.ActionUsersEdit, rbac.Target{}) {
		httpx.WriteError(w, http.StatusForbidden, "FORBIDDEN", "insufficient permissions")
		return
	}
	var req ReorderCompetenciesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_JSON", "invalid JSON")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION", err.Error())
		return
	}
	if err := h.svc.ReorderCompetencies(r.Context(), req.IDs); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) deleteCompetency(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.PrincipalFrom(r.Context())
	actor := rbac.Actor{UserID: p.UserID, Roles: p.Roles}
	if !rbac.Allow(actor, rbac.ActionUsersEdit, rbac.Target{}) {
		httpx.WriteError(w, http.StatusForbidden, "FORBIDDEN", "insufficient permissions")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "comp_id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_PARAM", "invalid comp_id")
		return
	}
	if err := h.svc.DeleteCompetency(r.Context(), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "competency not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listRequirements(w http.ResponseWriter, r *http.Request) {
	deptID, err := uuid.Parse(chi.URLParam(r, "dept_id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_PARAM", "invalid dept_id")
		return
	}
	reqs, err := h.svc.ListRequirements(r.Context(), deptID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}
	if reqs == nil {
		reqs = []Requirement{}
	}
	httpx.WriteJSON(w, http.StatusOK, reqs)
}

func (h *Handler) upsertRequirements(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.PrincipalFrom(r.Context())
	actor := rbac.Actor{UserID: p.UserID, Roles: p.Roles}
	deptID, err := uuid.Parse(chi.URLParam(r, "dept_id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_PARAM", "invalid dept_id")
		return
	}
	if !rbac.Allow(actor, rbac.ActionUsersEdit, rbac.Target{DepartmentID: &deptID}) {
		httpx.WriteError(w, http.StatusForbidden, "FORBIDDEN", "insufficient permissions")
		return
	}

	var reqs []UpsertRequirementRequest
	if err := json.NewDecoder(r.Body).Decode(&reqs); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_JSON", "invalid JSON")
		return
	}
	for i, req := range reqs {
		if err := h.validate.Struct(req); err != nil {
			httpx.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION", err.Error())
			return
		}
		_ = i
	}
	if err := h.svc.UpsertRequirements(r.Context(), deptID, reqs); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listPeriods(w http.ResponseWriter, r *http.Request) {
	var deptID *uuid.UUID
	if d := r.URL.Query().Get("department_id"); d != "" {
		id, err := uuid.Parse(d)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "BAD_PARAM", "invalid department_id")
			return
		}
		deptID = &id
	}
	periods, err := h.svc.ListPeriods(r.Context(), deptID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}
	if periods == nil {
		periods = []Period{}
	}
	httpx.WriteJSON(w, http.StatusOK, periods)
}

func (h *Handler) createPeriod(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.PrincipalFrom(r.Context())
	actor := rbac.Actor{UserID: p.UserID, Roles: p.Roles}
	if !rbac.Allow(actor, rbac.ActionUsersCreate, rbac.Target{}) {
		httpx.WriteError(w, http.StatusForbidden, "FORBIDDEN", "insufficient permissions")
		return
	}

	var req CreatePeriodRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_JSON", "invalid JSON")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION", err.Error())
		return
	}
	period, err := h.svc.CreatePeriod(r.Context(), req, p.UserID)
	if err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "INVALID", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, period)
}

func (h *Handler) getPeriod(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "period_id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_PARAM", "invalid period_id")
		return
	}
	period, scores, err := h.svc.GetPeriodWithScores(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "period not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}
	full, err := h.svc.GetPeriodFull(r.Context(), id)
	if err == nil {
		period = full
	}
	criteria, _ := h.svc.ListCriteria(r.Context(), id)
	assessees, _ := h.svc.ListAssessees(r.Context(), id)
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"period":    period,
		"scores":    scores,
		"criteria":  criteria,
		"assessees": assessees,
	})
}

func (h *Handler) upsertScore(w http.ResponseWriter, r *http.Request) {
	periodID, err := uuid.Parse(chi.URLParam(r, "period_id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_PARAM", "invalid period_id")
		return
	}

	var req UpsertScoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_JSON", "invalid JSON")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION", err.Error())
		return
	}

	p, _ := auth.PrincipalFrom(r.Context())
	score, err := h.svc.UpsertScore(r.Context(), periodID, req, p.UserID)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "period not found")
		case errors.Is(err, ErrNotParticipant):
			httpx.WriteError(w, http.StatusForbidden, "NOT_PARTICIPANT", "вы не назначены оценщиком в этом периоде")
		case errors.Is(err, ErrRoleNotInParticipant):
			httpx.WriteError(w, http.StatusForbidden, "WRONG_ROLE", "роль не соответствует вашему участию")
		default:
			httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", err.Error())
		}
		return
	}
	httpx.WriteJSON(w, http.StatusOK, score)
}

func (h *Handler) listEmployees(w http.ResponseWriter, r *http.Request) {
	deptID, err := uuid.Parse(chi.URLParam(r, "dept_id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_PARAM", "invalid dept_id")
		return
	}
	employees, err := h.svc.ListEmployees(r.Context(), deptID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}
	if employees == nil {
		employees = []Employee{}
	}
	httpx.WriteJSON(w, http.StatusOK, employees)
}

func (h *Handler) bulkUpsertScores(w http.ResponseWriter, r *http.Request) {
	periodID, err := uuid.Parse(chi.URLParam(r, "period_id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_PARAM", "invalid period_id")
		return
	}
	var reqs []UpsertScoreRequest
	if err := json.NewDecoder(r.Body).Decode(&reqs); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_JSON", "invalid JSON")
		return
	}
	for _, req := range reqs {
		if err := h.validate.Struct(req); err != nil {
			httpx.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION", err.Error())
			return
		}
	}
	p, _ := auth.PrincipalFrom(r.Context())
	if err := h.svc.BulkUpsertScores(r.Context(), periodID, reqs, p.UserID); err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "period not found")
		case errors.Is(err, ErrNotParticipant):
			httpx.WriteError(w, http.StatusForbidden, "NOT_PARTICIPANT", "вы не назначены оценщиком в этом периоде")
		case errors.Is(err, ErrRoleNotInParticipant):
			httpx.WriteError(w, http.StatusForbidden, "WRONG_ROLE", "роль не соответствует вашему участию")
		default:
			httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", err.Error())
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listParticipants(w http.ResponseWriter, r *http.Request) {
	periodID, err := uuid.Parse(chi.URLParam(r, "period_id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_PARAM", "invalid period_id")
		return
	}
	parts, err := h.svc.ListParticipants(r.Context(), periodID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, parts)
}

func (h *Handler) addParticipants(w http.ResponseWriter, r *http.Request) {
	periodID, err := uuid.Parse(chi.URLParam(r, "period_id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_PARAM", "invalid period_id")
		return
	}
	var req AddParticipantsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_JSON", err.Error())
		return
	}
	if err := h.validate.Struct(req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION", err.Error())
		return
	}
	if err := h.svc.AddParticipants(r.Context(), periodID, req.Participants); err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "period not found")
		case errors.Is(err, ErrParticipantsLocked):
			httpx.WriteError(w, http.StatusConflict, "LOCKED", "участники уже назначены и заблокированы")
		default:
			httpx.WriteError(w, http.StatusUnprocessableEntity, "INVALID", err.Error())
		}
		return
	}
	parts, _ := h.svc.ListParticipants(r.Context(), periodID)
	httpx.WriteJSON(w, http.StatusCreated, parts)
}

func (h *Handler) listMyPeriods(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.PrincipalFrom(r.Context())
	periods, err := h.svc.ListMyPeriods(r.Context(), p.UserID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, periods)
}

func (h *Handler) listMyScores(w http.ResponseWriter, r *http.Request) {
	periodID, err := uuid.Parse(chi.URLParam(r, "period_id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_PARAM", "invalid period_id")
		return
	}
	p, _ := auth.PrincipalFrom(r.Context())
	scores, err := h.svc.MyScoresIn(r.Context(), periodID, p.UserID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, scores)
}

func (h *Handler) listConsolidated(w http.ResponseWriter, r *http.Request) {
	periodID, err := uuid.Parse(chi.URLParam(r, "period_id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_PARAM", "invalid period_id")
		return
	}
	// Consolidated results are admin-only until the campaign is published
	// (FR-AS9/AS10). Non-admins may read them only after publication.
	p, _ := auth.PrincipalFrom(r.Context())
	if !p.HasRole("HR_ADMIN") {
		period, err := h.svc.GetPeriodFull(r.Context(), periodID)
		if err != nil || period.Status != StatusPublished {
			httpx.WriteError(w, http.StatusForbidden, "NOT_PUBLISHED", "результаты ещё не опубликованы")
			return
		}
	}
	rows, err := h.svc.ListConsolidated(r.Context(), periodID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, rows)
}

func (h *Handler) listUsersWithRole(w http.ResponseWriter, r *http.Request) {
	role := r.URL.Query().Get("role")
	if role == "" {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_PARAM", "role query param required")
		return
	}
	users, err := h.svc.ListUsersWithRole(r.Context(), role)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, users)
}
