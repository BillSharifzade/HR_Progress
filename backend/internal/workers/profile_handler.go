package workers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hrprogress/internal/httpx"
)

// mountProfile hangs the digital-profile stores off an existing
// /workers/{worker_id} route. Reads are open to any authenticated user, the
// same as the rest of a worker card; writes are HR_ADMIN, because this is
// personal data and the questionnaire is HR-owned.
func (h *Handler) mountProfile(r chi.Router) {
	r.Route("/profile", func(r chi.Router) {
		r.Get("/", h.getProfile)
		r.With(requireAdmin).Put("/", h.upsertProfile)
		r.With(requireAdmin).Delete("/", h.deleteProfile)
	})

	r.Route("/languages", func(r chi.Router) {
		r.Get("/", h.listLanguages)
		r.With(requireAdmin).Post("/", h.upsertLanguage)
		r.With(requireAdmin).Patch("/{language_id}", h.updateLanguage)
		r.With(requireAdmin).Delete("/{language_id}", h.deleteLanguage)
	})

	r.Route("/experience", func(r chi.Router) {
		r.Get("/", h.listExperience)
		r.With(requireAdmin).Post("/", h.createExperience)
		r.With(requireAdmin).Patch("/{experience_id}", h.updateExperience)
		r.With(requireAdmin).Delete("/{experience_id}", h.deleteExperience)
	})

	r.Route("/survey", func(r chi.Router) {
		r.Get("/", h.listSurvey)
		r.With(requireAdmin).Post("/", h.upsertSurvey)
		r.With(requireAdmin).Patch("/{answer_id}", h.updateSurvey)
		r.With(requireAdmin).Delete("/{answer_id}", h.deleteSurvey)
	})
}

// scopedIDs pulls the worker id and a child row id out of the URL in one go.
func scopedIDs(r *http.Request, param string) (worker, child uuid.UUID, err error) {
	worker, err = workerID(r)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	child, err = uuid.Parse(chi.URLParam(r, param))
	return worker, child, err
}

// decodeInto reads and validates a request body, writing the error response
// itself. A false return means the caller should simply stop.
func (h *Handler) decodeInto(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_JSON", err.Error())
		return false
	}
	if err := h.validate.Struct(dst); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION", err.Error())
		return false
	}
	return true
}

// writeResult collapses the repetitive (result, err) → response mapping.
func writeResult(w http.ResponseWriter, status int, body any, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "not found")
	case errors.Is(err, ErrInvalidInput):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION", err.Error())
	case err != nil:
		httpx.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
	default:
		httpx.WriteJSON(w, status, body)
	}
}

// --- profile --------------------------------------------------------------

func (h *Handler) getProfile(w http.ResponseWriter, r *http.Request) {
	id, err := workerID(r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "worker_id")
		return
	}
	p, err := h.repo.GetProfile(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		// Absent is normal — most people never filled the form in. Return an
		// empty profile so the UI renders editable blanks instead of an error.
		httpx.WriteJSON(w, http.StatusOK, Profile{
			UserID:                id,
			EducationLevels:       []string{},
			DevelopmentDirections: []string{},
			ProfessionalInterests: []string{},
			LearningFormats:       []string{},
		})
		return
	}
	writeResult(w, http.StatusOK, p, err)
}

func (h *Handler) upsertProfile(w http.ResponseWriter, r *http.Request) {
	id, err := workerID(r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "worker_id")
		return
	}
	var req UpsertProfileRequest
	if !h.decodeInto(w, r, &req) {
		return
	}
	p, err := h.repo.UpsertProfile(r.Context(), id, req)
	writeResult(w, http.StatusOK, p, err)
}

func (h *Handler) deleteProfile(w http.ResponseWriter, r *http.Request) {
	id, err := workerID(r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "worker_id")
		return
	}
	if err := h.repo.DeleteProfile(r.Context(), id); err != nil {
		writeResult(w, http.StatusNoContent, nil, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- languages ------------------------------------------------------------

func (h *Handler) listLanguages(w http.ResponseWriter, r *http.Request) {
	id, err := workerID(r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "worker_id")
		return
	}
	list, err := h.repo.ListLanguages(r.Context(), id)
	writeResult(w, http.StatusOK, list, err)
}

func (h *Handler) upsertLanguage(w http.ResponseWriter, r *http.Request) {
	id, err := workerID(r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "worker_id")
		return
	}
	var req UpsertLanguageRequest
	if !h.decodeInto(w, r, &req) {
		return
	}
	l, err := h.repo.UpsertLanguage(r.Context(), id, req)
	writeResult(w, http.StatusCreated, l, err)
}

func (h *Handler) updateLanguage(w http.ResponseWriter, r *http.Request) {
	worker, child, err := scopedIDs(r, "language_id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "id")
		return
	}
	var req UpsertLanguageRequest
	if !h.decodeInto(w, r, &req) {
		return
	}
	l, err := h.repo.UpdateLanguage(r.Context(), child, worker, req)
	writeResult(w, http.StatusOK, l, err)
}

func (h *Handler) deleteLanguage(w http.ResponseWriter, r *http.Request) {
	worker, child, err := scopedIDs(r, "language_id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "id")
		return
	}
	h.writeDelete(w, h.repo.DeleteLanguage(r.Context(), child, worker))
}

// --- work experience ------------------------------------------------------

func (h *Handler) listExperience(w http.ResponseWriter, r *http.Request) {
	id, err := workerID(r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "worker_id")
		return
	}
	list, err := h.repo.ListWorkExperience(r.Context(), id)
	writeResult(w, http.StatusOK, list, err)
}

func (h *Handler) createExperience(w http.ResponseWriter, r *http.Request) {
	id, err := workerID(r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "worker_id")
		return
	}
	var req UpsertWorkExperienceRequest
	if !h.decodeInto(w, r, &req) {
		return
	}
	e, err := h.repo.CreateWorkExperience(r.Context(), id, req)
	writeResult(w, http.StatusCreated, e, err)
}

func (h *Handler) updateExperience(w http.ResponseWriter, r *http.Request) {
	worker, child, err := scopedIDs(r, "experience_id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "id")
		return
	}
	var req UpsertWorkExperienceRequest
	if !h.decodeInto(w, r, &req) {
		return
	}
	e, err := h.repo.UpdateWorkExperience(r.Context(), child, worker, req)
	writeResult(w, http.StatusOK, e, err)
}

func (h *Handler) deleteExperience(w http.ResponseWriter, r *http.Request) {
	worker, child, err := scopedIDs(r, "experience_id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "id")
		return
	}
	h.writeDelete(w, h.repo.DeleteWorkExperience(r.Context(), child, worker))
}

// --- survey answers -------------------------------------------------------

func (h *Handler) listSurvey(w http.ResponseWriter, r *http.Request) {
	id, err := workerID(r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "worker_id")
		return
	}
	list, err := h.repo.ListSurveyAnswers(r.Context(), id)
	writeResult(w, http.StatusOK, list, err)
}

func (h *Handler) upsertSurvey(w http.ResponseWriter, r *http.Request) {
	id, err := workerID(r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "worker_id")
		return
	}
	var req UpsertSurveyAnswerRequest
	if !h.decodeInto(w, r, &req) {
		return
	}
	a, err := h.repo.UpsertSurveyAnswer(r.Context(), id, req)
	writeResult(w, http.StatusCreated, a, err)
}

func (h *Handler) updateSurvey(w http.ResponseWriter, r *http.Request) {
	worker, child, err := scopedIDs(r, "answer_id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "id")
		return
	}
	var req UpsertSurveyAnswerRequest
	if !h.decodeInto(w, r, &req) {
		return
	}
	a, err := h.repo.UpdateSurveyAnswer(r.Context(), child, worker, req)
	writeResult(w, http.StatusOK, a, err)
}

func (h *Handler) deleteSurvey(w http.ResponseWriter, r *http.Request) {
	worker, child, err := scopedIDs(r, "answer_id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "id")
		return
	}
	h.writeDelete(w, h.repo.DeleteSurveyAnswer(r.Context(), child, worker))
}

func (h *Handler) writeDelete(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "not found")
	case err != nil:
		httpx.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

// --- certification files --------------------------------------------------

func (h *Handler) updateCertification(w http.ResponseWriter, r *http.Request) {
	worker, cert, err := scopedIDs(r, "cert_id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "id")
		return
	}
	var req UpsertCertificationRequest
	if !h.decodeInto(w, r, &req) {
		return
	}
	c, err := h.repo.UpdateCertification(r.Context(), cert, worker, req)
	writeResult(w, http.StatusOK, c, err)
}

func (h *Handler) uploadCertificationFile(w http.ResponseWriter, r *http.Request) {
	worker, cert, err := scopedIDs(r, "cert_id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "id")
		return
	}
	if h.files == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "NO_STORAGE", "file storage is not configured")
		return
	}

	// Cap the request body before parsing so an oversized upload is rejected
	// without ever being buffered to disk by the multipart reader.
	r.Body = http.MaxBytesReader(w, r.Body, h.maxUploadBytes+(1<<20))
	file, header, err := r.FormFile("file")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_UPLOAD", err.Error())
		return
	}
	defer file.Close()

	relPath, contentType, size, err := h.files.Save(file, header.Filename)
	switch {
	case errors.Is(err, ErrUnsupportedType):
		httpx.WriteError(w, http.StatusUnsupportedMediaType, "UNSUPPORTED_TYPE", err.Error())
		return
	case errors.Is(err, ErrFileTooLarge):
		httpx.WriteError(w, http.StatusRequestEntityTooLarge, "TOO_LARGE", "file exceeds the size limit")
		return
	case err != nil:
		httpx.WriteError(w, http.StatusInternalServerError, "STORAGE_ERROR", err.Error())
		return
	}

	c, previous, err := h.repo.AttachCertificationFile(
		r.Context(), cert, worker, relPath, header.Filename, size, contentType)
	if err != nil {
		// The row never took the new file, so the bytes we just wrote are
		// orphaned — drop them rather than leaking disk.
		_ = h.files.Remove(relPath)
		writeResult(w, http.StatusOK, nil, err)
		return
	}
	// Only now is the replaced file unreferenced.
	if previous != nil {
		_ = h.files.Remove(*previous)
	}
	httpx.WriteJSON(w, http.StatusOK, c)
}

func (h *Handler) downloadCertificationFile(w http.ResponseWriter, r *http.Request) {
	worker, cert, err := scopedIDs(r, "cert_id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "id")
		return
	}
	c, err := h.repo.GetCertification(r.Context(), cert, worker)
	if errors.Is(err, ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "certification not found")
		return
	} else if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	if c.FilePath == nil || h.files == nil {
		httpx.WriteError(w, http.StatusNotFound, "NO_FILE", "no file attached")
		return
	}

	f, err := h.files.Open(*c.FilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			httpx.WriteError(w, http.StatusNotFound, "NO_FILE", "file is missing from storage")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "STORAGE_ERROR", err.Error())
		return
	}
	defer f.Close()

	name := ""
	if c.FileName != nil {
		name = *c.FileName
	}
	if c.ContentType != nil {
		w.Header().Set("Content-Type", *c.ContentType)
	}
	// Never let a stored document be interpreted as a page in our own origin.
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; sandbox")
	w.Header().Set("Content-Disposition", contentDisposition(name))
	if _, err := io.Copy(w, f); err != nil {
		return // response already partially written; nothing useful to say
	}
}

func (h *Handler) deleteCertificationFile(w http.ResponseWriter, r *http.Request) {
	worker, cert, err := scopedIDs(r, "cert_id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ID", "id")
		return
	}
	previous, err := h.repo.DetachCertificationFile(r.Context(), cert, worker)
	if err != nil {
		h.writeDelete(w, err)
		return
	}
	if previous != nil && h.files != nil {
		_ = h.files.Remove(*previous)
	}
	w.WriteHeader(http.StatusNoContent)
}
