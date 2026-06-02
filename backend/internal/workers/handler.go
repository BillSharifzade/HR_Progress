package workers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"hrprogress/internal/auth"
	"hrprogress/internal/httpx"
)

type Handler struct {
	repo     *Repository
	validate *validator.Validate
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{
		repo:     repo,
		validate: validator.New(validator.WithRequiredStructEnabled()),
	}
}

func (h *Handler) Mount(r chi.Router, jwt *auth.JWTIssuer) {
	r.Route("/workers", func(r chi.Router) {
		r.Use(auth.RequireAuth(jwt))
		r.Get("/", h.list)
		r.Post("/", h.create)
		r.Route("/{worker_id}", func(r chi.Router) {
			r.Get("/", h.get)
			r.Patch("/", h.update)
			r.Post("/activate", h.activate)
			r.Post("/deactivate", h.deactivate)
			r.Route("/certifications", func(r chi.Router) {
				r.Get("/", h.listCertifications)
				r.Post("/", h.createCertification)
				r.Delete("/{cert_id}", h.deleteCertification)
			})
			r.Route("/history", func(r chi.Router) {
				r.Get("/", h.listHistory)
				r.Post("/", h.createHistory)
			})
			r.Route("/roles", func(r chi.Router) {
				r.Get("/", h.listRoles)
				r.With(requireAdmin).Post("/", h.grantRole)
				r.With(requireAdmin).Delete("/{assignment_id}", h.revokeRole)
			})
			r.With(requireAdmin).Post("/credentials/reset", h.resetCredentials)
		})
	})

	r.Route("/positions", func(r chi.Router) {
		r.Use(auth.RequireAuth(jwt))
		r.Get("/", h.listPositions)
		r.Post("/", h.createPosition)
	})

	r.Route("/sections", func(r chi.Router) {
		r.Use(auth.RequireAuth(jwt))
		r.Get("/", h.listSections)
		r.Post("/", h.createSection)
		r.Patch("/{section_id}", h.updateSection)
		r.Delete("/{section_id}", h.deleteSection)
	})
}

func workerID(r *http.Request) (uuid.UUID, error) {
	return uuid.Parse(chi.URLParam(r, "worker_id"))
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

func (h *Handler) listRoles(w http.ResponseWriter, r *http.Request) {
	id, err := workerID(r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "worker_id")
		return
	}
	list, err := h.repo.ListRoleAssignments(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, list)
}

func (h *Handler) grantRole(w http.ResponseWriter, r *http.Request) {
	id, err := workerID(r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "worker_id")
		return
	}
	var req GrantRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_JSON", err.Error())
		return
	}
	if err := h.validate.Struct(req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION", err.Error())
		return
	}
	actor, _ := auth.PrincipalFrom(r.Context())
	assignment, err := h.repo.GrantRole(r.Context(), id, req, actor.UserID)
	switch {
	case errors.Is(err, ErrInvalidScope):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "INVALID_SCOPE", "role requires a specific scope")
		return
	case errors.Is(err, ErrRoleExists):
		httpx.WriteError(w, http.StatusConflict, "ROLE_EXISTS", "role already granted for that scope")
		return
	case err != nil:
		httpx.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, assignment)
}

func (h *Handler) resetCredentials(w http.ResponseWriter, r *http.Request) {
	id, err := workerID(r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "worker_id")
		return
	}
	// Always generate server-side — admins can't propose a password.
	// This guarantees every reset yields a fresh, high-entropy credential
	// regardless of what the client may have sent in the body (e.g.
	// browser-autofilled fields).
	password, err := auth.GenerateTempPassword()
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "RAND_ERROR", err.Error())
		return
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "HASH_ERROR", err.Error())
		return
	}
	username, err := h.repo.ResetPassword(r.Context(), id, hash)
	if errors.Is(err, ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "worker not found")
		return
	} else if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{
		"username": username,
		"password": password,
	})
}

func (h *Handler) revokeRole(w http.ResponseWriter, r *http.Request) {
	id, err := workerID(r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "worker_id")
		return
	}
	assignmentID, err := uuid.Parse(chi.URLParam(r, "assignment_id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "assignment_id")
		return
	}
	if err := h.repo.RevokeRole(r.Context(), id, assignmentID); errors.Is(err, ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "role assignment not found")
		return
	} else if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	f := ListFilter{}
	if v := r.URL.Query().Get("department_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "INVALID_PARAM", "department_id")
			return
		}
		f.DepartmentID = &id
	}
	if v := r.URL.Query().Get("section_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "INVALID_PARAM", "section_id")
			return
		}
		f.SectionID = &id
	}
	if v := r.URL.Query().Get("grade_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "INVALID_PARAM", "grade_id")
			return
		}
		f.GradeID = &id
	}
	f.Search = r.URL.Query().Get("search")
	f.IncludeInactive = r.URL.Query().Get("include_inactive") == "true"

	list, err := h.repo.List(r.Context(), f)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	if list == nil {
		list = []WorkerSummary{}
	}
	httpx.WriteJSON(w, http.StatusOK, list)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, err := workerID(r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "worker_id")
		return
	}
	worker, err := h.repo.Get(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "worker not found")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, worker)
}

func (h *Handler) listCertifications(w http.ResponseWriter, r *http.Request) {
	id, err := workerID(r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "worker_id")
		return
	}
	list, err := h.repo.ListCertifications(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	if list == nil {
		list = []Certification{}
	}
	httpx.WriteJSON(w, http.StatusOK, list)
}

func (h *Handler) createCertification(w http.ResponseWriter, r *http.Request) {
	id, err := workerID(r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "worker_id")
		return
	}
	var req UpsertCertificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_JSON", err.Error())
		return
	}
	if err := h.validate.Struct(req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION", err.Error())
		return
	}
	cert, err := h.repo.CreateCertification(r.Context(), id, req)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, cert)
}

func (h *Handler) deleteCertification(w http.ResponseWriter, r *http.Request) {
	workerUUID, err := workerID(r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "worker_id")
		return
	}
	certUUID, err := uuid.Parse(chi.URLParam(r, "cert_id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "cert_id")
		return
	}
	if err := h.repo.DeleteCertification(r.Context(), certUUID, workerUUID); errors.Is(err, ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "certification not found")
		return
	} else if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listHistory(w http.ResponseWriter, r *http.Request) {
	id, err := workerID(r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "worker_id")
		return
	}
	list, err := h.repo.ListHistory(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	if list == nil {
		list = []History{}
	}
	httpx.WriteJSON(w, http.StatusOK, list)
}

func (h *Handler) createHistory(w http.ResponseWriter, r *http.Request) {
	id, err := workerID(r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "worker_id")
		return
	}
	actor, _ := auth.PrincipalFrom(r.Context())

	var req CreateHistoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_JSON", err.Error())
		return
	}
	if err := h.validate.Struct(req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION", err.Error())
		return
	}
	h_, err := h.repo.CreateHistory(r.Context(), id, req, actor.UserID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, h_)
}

func (h *Handler) listPositions(w http.ResponseWriter, r *http.Request) {
	list, err := h.repo.ListPositions(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	if list == nil {
		list = []Position{}
	}
	httpx.WriteJSON(w, http.StatusOK, list)
}

func (h *Handler) createPosition(w http.ResponseWriter, r *http.Request) {
	var req CreatePositionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_JSON", err.Error())
		return
	}
	if err := h.validate.Struct(req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION", err.Error())
		return
	}
	p, err := h.repo.CreatePosition(r.Context(), req.Name)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, p)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req CreateWorkerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_JSON", err.Error())
		return
	}
	if err := h.validate.Struct(req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION", err.Error())
		return
	}
	// Always generate the initial password server-side so the admin never
	// types it (avoids browser-autofill repeats and weak inputs).
	password, err := auth.GenerateTempPassword()
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "RAND_ERROR", err.Error())
		return
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "HASH_ERROR", err.Error())
		return
	}
	worker, err := h.repo.Create(r.Context(), req, hash)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{
		"worker":   worker,
		"username": worker.Username,
		"password": password,
	})
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, err := workerID(r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "worker_id")
		return
	}
	var req UpdateWorkerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_JSON", err.Error())
		return
	}
	if err := h.validate.Struct(req); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION", err.Error())
		return
	}
	worker, err := h.repo.Update(r.Context(), id, req)
	if errors.Is(err, ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "worker not found")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, worker)
}

func (h *Handler) activate(w http.ResponseWriter, r *http.Request) {
	id, err := workerID(r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "worker_id")
		return
	}
	if err := h.repo.SetActive(r.Context(), id, true); errors.Is(err, ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "worker not found")
		return
	} else if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) deactivate(w http.ResponseWriter, r *http.Request) {
	id, err := workerID(r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "worker_id")
		return
	}
	if err := h.repo.SetActive(r.Context(), id, false); errors.Is(err, ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "worker not found")
		return
	} else if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listSections(w http.ResponseWriter, r *http.Request) {
	var deptID *uuid.UUID
	if v := r.URL.Query().Get("department_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "INVALID_PARAM", "department_id")
			return
		}
		deptID = &id
	}
	list, err := h.repo.ListSections(r.Context(), deptID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	if list == nil {
		list = []Section{}
	}
	httpx.WriteJSON(w, http.StatusOK, list)
}

func (h *Handler) createSection(w http.ResponseWriter, r *http.Request) {
	var body struct {
		DepartmentID string  `json:"department_id"`
		Name         string  `json:"name"`
		Description  *string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_BODY", err.Error())
		return
	}
	deptID, err := uuid.Parse(body.DepartmentID)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_PARAM", "department_id")
		return
	}
	if body.Name == "" {
		httpx.WriteError(w, http.StatusBadRequest, "VALIDATION", "name is required")
		return
	}
	s, err := h.repo.CreateSection(r.Context(), CreateSectionRequest{
		DepartmentID: deptID,
		Name:         body.Name,
		Description:  body.Description,
	})
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, s)
}

func (h *Handler) updateSection(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "section_id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_PARAM", "section_id")
		return
	}
	var body struct {
		Name        string  `json:"name"`
		Description *string `json:"description"`
		IsActive    bool    `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_BODY", err.Error())
		return
	}
	s, err := h.repo.UpdateSection(r.Context(), id, UpdateSectionRequest{
		Name:        body.Name,
		Description: body.Description,
		IsActive:    body.IsActive,
	})
	if errors.Is(err, ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "section not found")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s)
}

func (h *Handler) deleteSection(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "section_id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_PARAM", "section_id")
		return
	}
	if err := h.repo.DeleteSection(r.Context(), id); errors.Is(err, ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "section not found")
		return
	} else if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
