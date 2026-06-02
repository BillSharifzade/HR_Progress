package onef

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"hrprogress/internal/auth"
	"hrprogress/internal/workers"
)

type Service struct {
	repo        *Repository
	client      *Client
	workersRepo *workers.Repository
	log         *slog.Logger

	mu      sync.Mutex // serialize concurrent RunSync calls (manual + cron overlap)
	running bool
}

func NewService(repo *Repository, client *Client, workersRepo *workers.Repository, log *slog.Logger) *Service {
	return &Service{repo: repo, client: client, workersRepo: workersRepo, log: log}
}

// Configured reports whether the underlying 1F client has a base URL.
func (s *Service) Configured() bool { return s.client.Configured() }

// RunSync polls 1F once, reconciles workers, and records a sync_run row.
// triggeredBy may be nil for cron-driven runs.
func (s *Service) RunSync(ctx context.Context, trigger string, triggeredBy *uuid.UUID) (*SyncResult, error) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil, errors.New("sync already in progress")
	}
	s.running = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	if !s.client.Configured() {
		return nil, errors.New("1F endpoint not configured (set ONEF_BASE_URL)")
	}

	started := time.Now()
	runID, err := s.repo.StartRun(ctx, trigger, triggeredBy)
	if err != nil {
		return nil, fmt.Errorf("start run: %w", err)
	}

	res := SyncResult{RunID: runID, Status: StatusRunning}

	users, err := s.client.FetchUsers(ctx)
	if err != nil {
		res.DurationMS = int(time.Since(started).Milliseconds())
		res.Status = StatusFailed
		res.Error = err.Error()
		_ = s.repo.FinishRun(ctx, runID, res, err.Error())
		return &res, err
	}
	res.FetchedCount = len(users)

	for _, u := range users {
		if u.UserID <= 0 {
			res.SkippedCount++
			s.log.Warn("1F user missing UserID", slog.Any("user", u))
			continue
		}
		if isIgnoredDepartment(u.Department) {
			// Out-of-scope department per product policy. Don't touch the DB,
			// don't log — just tally it so admins see the filter is working.
			res.SkippedCount++
			continue
		}
		action, err := s.reconcileOne(ctx, u)
		if err != nil {
			res.SkippedCount++
			s.log.Error("1F reconcile failed",
				slog.Int64("one_f_user_id", u.UserID),
				slog.String("err", err.Error()))
			continue
		}
		switch action {
		case "created":
			res.CreatedCount++
		case "updated":
			res.UpdatedCount++
		case "noop":
			// updated counter still bumps so admins see activity
			res.UpdatedCount++
		}
	}

	res.DurationMS = int(time.Since(started).Milliseconds())
	res.Status = StatusSuccess
	if err := s.repo.FinishRun(ctx, runID, res, ""); err != nil {
		s.log.Error("1F finish run", slog.String("err", err.Error()))
	}
	return &res, nil
}

// reconcileOne returns "created" | "updated" on success.
func (s *Service) reconcileOne(ctx context.Context, u OneFUser) (string, error) {
	// Resolve dept + section (auto-create on first sight).
	deptName := strings.TrimSpace(u.Department)
	if deptName == "" {
		return "", errors.New("missing Department")
	}
	deptID, err := s.repo.FindOrCreateDepartment(ctx, deptName)
	if err != nil {
		return "", fmt.Errorf("dept %q: %w", deptName, err)
	}

	var sectionIDPtr *uuid.UUID
	if otdel := strings.TrimSpace(u.Otdel); otdel != "" {
		sid, err := s.repo.FindOrCreateSection(ctx, deptID, otdel)
		if err != nil {
			return "", fmt.Errorf("section %q: %w", otdel, err)
		}
		sectionIDPtr = &sid
	}

	gradeID, err := s.repo.GradeIDByLevel(ctx, InferGradeLevel(u.Dolzhnost))
	if err != nil {
		return "", fmt.Errorf("grade lookup: %w", err)
	}

	birthDate := parseISODate(u.BirthDate)
	hiredAt := parseDottedDate(u.DateHired)
	fullName := strings.TrimSpace(u.DisplayName)
	if fullName == "" {
		fullName = strings.TrimSpace(u.LastName + " " + u.FirstName)
	}

	emailPtr := optionalString(u.Email)
	phonePtr := optionalString(u.PhoneNumber)
	positionPtr := optionalString(u.Dolzhnost)
	var managerPtr *int64
	if u.Manager > 0 {
		m := u.Manager
		managerPtr = &m
	}
	deptIDPtr := &deptID

	match, err := s.repo.FindByOneFUserID(ctx, u.UserID)
	if err != nil {
		return "", fmt.Errorf("lookup: %w", err)
	}

	var userID uuid.UUID
	var action string
	if match == nil {
		username, err := s.uniqueUsername(ctx, fullName)
		if err != nil {
			return "", fmt.Errorf("username: %w", err)
		}
		pwd, err := auth.GenerateTempPassword()
		if err != nil {
			return "", fmt.Errorf("password gen: %w", err)
		}
		hash, err := auth.HashPassword(pwd)
		if err != nil {
			return "", fmt.Errorf("password hash: %w", err)
		}
		userID, err = s.repo.InsertWorker(ctx, CreateWorkerInput{
			Username:          username,
			PasswordHash:      hash,
			FullName:          fullName,
			Email:             emailPtr,
			PhoneNumber:       phonePtr,
			BirthDate:         birthDate,
			HiredAt:           hiredAt,
			DepartmentID:      deptIDPtr,
			SectionID:         sectionIDPtr,
			GradeID:           gradeID,
			Position:          positionPtr,
			OneFUserID:        u.UserID,
			OneFManagerUserID: managerPtr,
			OneFIsManager:     u.IsManager == 1,
		})
		if err != nil {
			return "", fmt.Errorf("insert: %w", err)
		}
		action = "created"
	} else {
		err := s.repo.UpdateWorker(ctx, match.ID, UpdateWorkerInput{
			FullName:          fullName,
			Email:             emailPtr,
			PhoneNumber:       phonePtr,
			BirthDate:         birthDate,
			HiredAt:           hiredAt,
			DepartmentID:      deptIDPtr,
			SectionID:         sectionIDPtr,
			Position:          positionPtr,
			OneFManagerUserID: managerPtr,
			OneFIsManager:     u.IsManager == 1,
			GradeID:           gradeID,
		})
		if err != nil {
			return "", fmt.Errorf("update: %w", err)
		}
		userID = match.ID
		action = "updated"
	}

	// Manager role derivation:
	//   IsManager=1 + Otdel set  → SECTION_HEAD scoped to that section
	//   IsManager=1 + Otdel null → DEPT_HEAD scoped to dept
	// Granted idempotently; never revoked (HR admin owns role removal).
	if u.IsManager == 1 {
		if sectionIDPtr != nil {
			if err := s.repo.GrantSectionHead(ctx, userID, *sectionIDPtr, deptID); err != nil {
				s.log.Error("grant SECTION_HEAD",
					slog.String("user_id", userID.String()),
					slog.String("err", err.Error()))
			}
		} else {
			if err := s.repo.GrantDeptHead(ctx, userID, deptID); err != nil {
				s.log.Error("grant DEPT_HEAD",
					slog.String("user_id", userID.String()),
					slog.String("err", err.Error()))
			}
		}
	}

	return action, nil
}

func (s *Service) ListRuns(ctx context.Context, limit int) ([]SyncRun, error) {
	return s.repo.ListRuns(ctx, limit)
}

// --- helpers ---

func (s *Service) uniqueUsername(ctx context.Context, fullName string) (string, error) {
	base := workers.UsernameBase(fullName)
	if base == "" {
		base = "user"
	}
	return s.workersRepo.UniqueUsername(ctx, base)
}

func parseISODate(s string) *time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil
	}
	return &t
}

func parseDottedDate(s string) *time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	t, err := time.Parse("02.01.2006", s)
	if err != nil {
		return nil
	}
	return &t
}

func optionalString(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}
