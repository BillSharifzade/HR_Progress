package competency

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hrprogress/internal/auth"
	"hrprogress/internal/httpx"
)

// optUUID parses an optional uuid query param. Empty → nil, invalid → error.
func optUUID(r *http.Request, key string) (*uuid.UUID, error) {
	v := r.URL.Query().Get(key)
	if v == "" {
		return nil, nil
	}
	id, err := uuid.Parse(v)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// ── Criteria (FR-AS3) ────────────────────────────────────────────────────────

func (h *Handler) listCriteria(w http.ResponseWriter, r *http.Request) {
	periodID, err := uuid.Parse(chi.URLParam(r, "period_id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_PARAM", "invalid period_id")
		return
	}
	out, err := h.svc.ListCriteria(r.Context(), periodID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) setCriteria(w http.ResponseWriter, r *http.Request) {
	periodID, err := uuid.Parse(chi.URLParam(r, "period_id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_PARAM", "invalid period_id")
		return
	}
	var req SetCriteriaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_JSON", "invalid JSON")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION", err.Error())
		return
	}
	if err := h.svc.SetCriteria(r.Context(), periodID, req.Criteria); err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "period not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	out, _ := h.svc.ListCriteria(r.Context(), periodID)
	httpx.WriteJSON(w, http.StatusOK, out)
}

// ── Assessees (FR-AS2) ───────────────────────────────────────────────────────

func (h *Handler) listAssessees(w http.ResponseWriter, r *http.Request) {
	periodID, err := uuid.Parse(chi.URLParam(r, "period_id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_PARAM", "invalid period_id")
		return
	}
	out, err := h.svc.ListAssessees(r.Context(), periodID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) addAssessees(w http.ResponseWriter, r *http.Request) {
	periodID, err := uuid.Parse(chi.URLParam(r, "period_id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_PARAM", "invalid period_id")
		return
	}
	var req AddAssesseesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_JSON", "invalid JSON")
		return
	}
	added, err := h.svc.AddAssessees(r.Context(), periodID, req)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "period not found")
			return
		}
		httpx.WriteError(w, http.StatusUnprocessableEntity, "INVALID", err.Error())
		return
	}
	out, _ := h.svc.ListAssessees(r.Context(), periodID)
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"added": added, "assessees": out})
}

func (h *Handler) removeAssessee(w http.ResponseWriter, r *http.Request) {
	periodID, err := uuid.Parse(chi.URLParam(r, "period_id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_PARAM", "invalid period_id")
		return
	}
	userID, err := uuid.Parse(chi.URLParam(r, "user_id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_PARAM", "invalid user_id")
		return
	}
	if err := h.svc.RemoveAssessee(r.Context(), periodID, userID); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Per-assessee assessors (FR-AS4) ──────────────────────────────────────────

func (h *Handler) listAssesseeAssessors(w http.ResponseWriter, r *http.Request) {
	periodID, err := uuid.Parse(chi.URLParam(r, "period_id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_PARAM", "invalid period_id")
		return
	}
	out, err := h.svc.ListAssesseeAssessors(r.Context(), periodID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) setAssesseeAssessors(w http.ResponseWriter, r *http.Request) {
	periodID, err := uuid.Parse(chi.URLParam(r, "period_id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_PARAM", "invalid period_id")
		return
	}
	assesseeID, err := uuid.Parse(chi.URLParam(r, "user_id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_PARAM", "invalid user_id")
		return
	}
	var req SetAssessorsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_JSON", "invalid JSON")
		return
	}
	if err := h.svc.SetAssesseeAssessors(r.Context(), periodID, assesseeID, req.AssessorUserIDs); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "INVALID", err.Error())
		return
	}
	out, _ := h.svc.ListAssesseeAssessors(r.Context(), periodID)
	httpx.WriteJSON(w, http.StatusOK, out)
}

// ── Lifecycle (Section 5, FR-AS9, FR-AS10) ───────────────────────────────────

func (h *Handler) transitionPeriod(w http.ResponseWriter, r *http.Request) {
	periodID, err := uuid.Parse(chi.URLParam(r, "period_id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_PARAM", "invalid period_id")
		return
	}
	var body struct {
		To string `json:"to"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_JSON", "invalid JSON")
		return
	}
	p, _ := auth.PrincipalFrom(r.Context())
	period, err := h.svc.Transition(r.Context(), periodID, body.To, p.UserID)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "period not found")
		case errors.Is(err, ErrBadTransition):
			httpx.WriteError(w, http.StatusConflict, "BAD_TRANSITION", err.Error())
		case errors.Is(err, ErrNoCriteria):
			httpx.WriteError(w, http.StatusUnprocessableEntity, "NO_CRITERIA", "не заданы критерии оценки")
		case errors.Is(err, ErrNoAssessees):
			httpx.WriteError(w, http.StatusUnprocessableEntity, "NO_ASSESSEES", "не назначены участники")
		default:
			httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL", err.Error())
		}
		return
	}
	httpx.WriteJSON(w, http.StatusOK, period)
}

// ── Learning groups (FR-AS13) ────────────────────────────────────────────────

func (h *Handler) listGroups(w http.ResponseWriter, r *http.Request) {
	periodID, err := uuid.Parse(chi.URLParam(r, "period_id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_PARAM", "invalid period_id")
		return
	}
	// Groups are visible to Admin and AtS/Отдел развития (Section 6).
	p, _ := auth.PrincipalFrom(r.Context())
	if !p.HasRole("HR_ADMIN") && !p.HasRole("ATS") {
		httpx.WriteError(w, http.StatusForbidden, "FORBIDDEN", "только Администратор или AtS")
		return
	}
	out, err := h.svc.ListGroups(r.Context(), periodID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) listGroupJournal(w http.ResponseWriter, r *http.Request) {
	periodID, err := uuid.Parse(chi.URLParam(r, "period_id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_PARAM", "invalid period_id")
		return
	}
	p, _ := auth.PrincipalFrom(r.Context())
	if !p.HasRole("HR_ADMIN") && !p.HasRole("ATS") {
		httpx.WriteError(w, http.StatusForbidden, "FORBIDDEN", "только Администратор или AtS")
		return
	}
	out, err := h.svc.ListGroupJournal(r.Context(), periodID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) regenerateGroups(w http.ResponseWriter, r *http.Request) {
	periodID, err := uuid.Parse(chi.URLParam(r, "period_id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_PARAM", "invalid period_id")
		return
	}
	var body struct {
		GroupSize *int `json:"group_size"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	p, _ := auth.PrincipalFrom(r.Context())
	if err := h.svc.RegenerateGroups(r.Context(), periodID, body.GroupSize, p.UserID); err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "period not found")
			return
		}
		httpx.WriteError(w, http.StatusUnprocessableEntity, "INVALID", err.Error())
		return
	}
	out, _ := h.svc.ListGroups(r.Context(), periodID)
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) moveGroupMember(w http.ResponseWriter, r *http.Request) {
	periodID, err := uuid.Parse(chi.URLParam(r, "period_id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_PARAM", "invalid period_id")
		return
	}
	var req MoveMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_JSON", "invalid JSON")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION", err.Error())
		return
	}
	p, _ := auth.PrincipalFrom(r.Context())
	if err := h.svc.MoveMember(r.Context(), periodID, req.UserID, req.ToGroupID, p.UserID); err != nil {
		switch {
		case errors.Is(err, ErrMemberNotInPeriod):
			httpx.WriteError(w, http.StatusNotFound, "NOT_MEMBER", "сотрудник не входит в группы кампании")
		case errors.Is(err, ErrNotFound):
			httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "целевая группа не найдена")
		default:
			httpx.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		}
		return
	}
	out, _ := h.svc.ListGroups(r.Context(), periodID)
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) confirmGroups(w http.ResponseWriter, r *http.Request) {
	periodID, err := uuid.Parse(chi.URLParam(r, "period_id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_PARAM", "invalid period_id")
		return
	}
	p, _ := auth.PrincipalFrom(r.Context())
	if err := h.svc.ConfirmGroups(r.Context(), periodID, p.UserID); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	out, _ := h.svc.ListGroups(r.Context(), periodID)
	httpx.WriteJSON(w, http.StatusOK, out)
}

// ── Interpretation reference (FR-AS7.2) ──────────────────────────────────────

func (h *Handler) listInterpretations(w http.ResponseWriter, r *http.Request) {
	deptID, err := optUUID(r, "department_id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_PARAM", "invalid department_id")
		return
	}
	gradeID, err := optUUID(r, "grade_id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_PARAM", "invalid grade_id")
		return
	}
	compID, err := optUUID(r, "competency_id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_PARAM", "invalid competency_id")
		return
	}
	out, err := h.svc.ListInterpretations(r.Context(), deptID, gradeID, compID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) lookupInterpretation(w http.ResponseWriter, r *http.Request) {
	assesseeID, err := uuid.Parse(r.URL.Query().Get("assessee_id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_PARAM", "invalid assessee_id")
		return
	}
	compID, err := uuid.Parse(r.URL.Query().Get("competency_id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_PARAM", "invalid competency_id")
		return
	}
	score, err := parseScore(r.URL.Query().Get("score"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_PARAM", "score must be 1..10")
		return
	}
	out, err := h.svc.LookupInterpretationForScore(r.Context(), assesseeID, compID, score)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) upsertInterpretation(w http.ResponseWriter, r *http.Request) {
	var req UpsertInterpretationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_JSON", "invalid JSON")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION", err.Error())
		return
	}
	p, _ := auth.PrincipalFrom(r.Context())
	out, err := h.svc.UpsertInterpretation(r.Context(), req, p.UserID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) deleteInterpretation(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_PARAM", "invalid id")
		return
	}
	p, _ := auth.PrincipalFrom(r.Context())
	if err := h.svc.DeleteInterpretation(r.Context(), id, p.UserID); err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "interpretation not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) copyInterpretations(w http.ResponseWriter, r *http.Request) {
	var req CopyInterpretationsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_JSON", "invalid JSON")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION", err.Error())
		return
	}
	p, _ := auth.PrincipalFrom(r.Context())
	n, err := h.svc.CopyInterpretations(r.Context(), req, p.UserID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"copied": n})
}

func (h *Handler) interpretationHistory(w http.ResponseWriter, r *http.Request) {
	deptID, _ := optUUID(r, "department_id")
	gradeID, _ := optUUID(r, "grade_id")
	compID, _ := optUUID(r, "competency_id")
	var scorePtr *int
	if sv := r.URL.Query().Get("score"); sv != "" {
		s, err := parseScore(sv)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "BAD_PARAM", "score must be 1..10")
			return
		}
		scorePtr = &s
	}
	out, err := h.svc.InterpretationHistory(r.Context(), deptID, gradeID, compID, scorePtr)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

// ── Worker results (FR-AS10, AS11) ───────────────────────────────────────────

func (h *Handler) myAssessmentResults(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.PrincipalFrom(r.Context())
	out, err := h.svc.MyPublishedResults(r.Context(), p.UserID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func parseScore(s string) (int, error) {
	if s == "" {
		return 0, errors.New("empty")
	}
	n := 0
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0, errors.New("not a number")
		}
		n = n*10 + int(ch-'0')
	}
	if n < 1 || n > 10 {
		return 0, errors.New("out of range")
	}
	return n, nil
}
