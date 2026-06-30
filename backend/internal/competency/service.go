package competency

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("not found")

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListDepartments(ctx context.Context) ([]Department, error) {
	return s.repo.ListDepartments(ctx)
}

func (s *Service) ListEmployees(ctx context.Context, deptID uuid.UUID) ([]Employee, error) {
	return s.repo.ListEmployees(ctx, deptID)
}

func (s *Service) BulkUpsertScores(ctx context.Context, periodID uuid.UUID, reqs []UpsertScoreRequest, principalID uuid.UUID) error {
	if _, err := s.repo.GetPeriod(ctx, periodID); err != nil {
		return ErrNotFound
	}
	roles, err := s.repo.MyRolesIn(ctx, periodID, principalID)
	if err != nil {
		return err
	}
	if len(roles) == 0 {
		return ErrNotParticipant
	}
	roleSet := make(map[string]bool, len(roles))
	for _, r := range roles {
		roleSet[r] = true
	}
	for i, req := range reqs {
		role := req.AssessorRole
		if role == "" {
			if len(roles) != 1 {
				return fmt.Errorf("score #%d: assessor_role is required when participant holds multiple roles", i)
			}
			role = roles[0]
		}
		if !roleSet[role] {
			return ErrRoleNotInParticipant
		}
		autoInterp := s.computeAutoInterp(ctx, req.EmployeeID, req.CompetencyID, req.Score)
		if _, err := s.repo.UpsertScoreFor(ctx, periodID, req.EmployeeID, req.CompetencyID, principalID, role, req.Score, req.Feedback, autoInterp); err != nil {
			return err
		}
		if err := s.repo.MaybeFinalize(ctx, periodID, req.EmployeeID, req.CompetencyID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) AddParticipants(ctx context.Context, periodID uuid.UUID, parts []ParticipantInput) error {
	if _, err := s.repo.GetPeriod(ctx, periodID); err != nil {
		return ErrNotFound
	}
	// HEAD / DEPT_HEAD / DCR_HEAD evaluators are derived from organizational
	// structure (user_roles SECTION_HEAD + DEPT_HEAD scopes) at scoring time,
	// not stored as participants. HRA is the consolidated assessor average,
	// not a person. So the only role admins assign here is ASSESSOR.
	roleCounts := map[string]int{}
	for _, p := range parts {
		roleCounts[p.Role]++
	}
	if roleCounts["ASSESSOR"] < 2 {
		return errors.New("at least 2 ASSESSOR participants required")
	}
	return s.repo.AddParticipants(ctx, periodID, parts)
}

func (s *Service) ListParticipants(ctx context.Context, periodID uuid.UUID) ([]Participant, error) {
	return s.repo.ListParticipants(ctx, periodID)
}

func (s *Service) ListMyPeriods(ctx context.Context, userID uuid.UUID) ([]MyPeriod, error) {
	return s.repo.ListMyPeriods(ctx, userID)
}

func (s *Service) MyScoresIn(ctx context.Context, periodID, userID uuid.UUID) ([]Score, error) {
	return s.repo.MyScoresIn(ctx, periodID, userID)
}

func (s *Service) ListConsolidated(ctx context.Context, periodID uuid.UUID) ([]ConsolidatedScore, error) {
	return s.repo.ListConsolidated(ctx, periodID)
}

func (s *Service) ListUsersWithRole(ctx context.Context, role string) ([]Employee, error) {
	return s.repo.ListUsersWithRole(ctx, role)
}

func (s *Service) ListAllDepartments(ctx context.Context) ([]Department, error) {
	return s.repo.ListAllDepartments(ctx)
}

func (s *Service) ListGrades(ctx context.Context) ([]Grade, error) {
	return s.repo.ListGrades(ctx)
}

func (s *Service) CreateDepartment(ctx context.Context, req CreateDepartmentRequest) (Department, error) {
	code, err := s.uniqueDeptCode(ctx, deriveDeptCode(req.Name))
	if err != nil {
		return Department{}, err
	}
	return s.repo.CreateDepartment(ctx, code, req.Name, req.Description)
}

func (s *Service) uniqueDeptCode(ctx context.Context, base string) (string, error) {
	if base == "" {
		base = "DEP"
	}
	candidate := base
	for i := 2; ; i++ {
		exists, err := s.repo.DepartmentCodeExists(ctx, candidate)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s%d", base, i)
		if i > 999 {
			return "", errors.New("could not generate unique department code")
		}
	}
}

func (s *Service) UpdateDepartment(ctx context.Context, id uuid.UUID, req UpdateDepartmentRequest) (Department, error) {
	return s.repo.UpdateDepartment(ctx, id, req)
}

func (s *Service) DeleteDepartment(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteDepartment(ctx, id)
}

func (s *Service) ListCompetencies(ctx context.Context) ([]Competency, error) {
	return s.repo.ListCompetencies(ctx)
}

func (s *Service) ListRequirements(ctx context.Context, deptID uuid.UUID) ([]Requirement, error) {
	return s.repo.ListRequirements(ctx, deptID)
}

func (s *Service) UpsertRequirements(ctx context.Context, deptID uuid.UUID, reqs []UpsertRequirementRequest) error {
	return s.repo.UpsertRequirements(ctx, deptID, reqs)
}

func (s *Service) ListPeriods(ctx context.Context, deptID *uuid.UUID) ([]Period, error) {
	return s.repo.ListPeriods(ctx, deptID)
}

func (s *Service) CreatePeriod(ctx context.Context, req CreatePeriodRequest, createdBy uuid.UUID) (Period, error) {
	start, err := time.Parse("2006-01-02", req.PeriodStart)
	if err != nil {
		return Period{}, errors.New("invalid period_start date, expected YYYY-MM-DD")
	}
	end, err := time.Parse("2006-01-02", req.PeriodEnd)
	if err != nil {
		return Period{}, errors.New("invalid period_end date, expected YYYY-MM-DD")
	}
	if !end.After(start) {
		return Period{}, errors.New("period_end must be after period_start")
	}

	p := Period{
		Title:       req.Title,
		PeriodStart: start,
		PeriodEnd:   end,
		IsActive:    true,
		CreatedBy:   &createdBy,
		GroupSize:   12,
	}
	if req.GroupSize != nil && *req.GroupSize > 0 {
		p.GroupSize = *req.GroupSize
	}
	if req.DepartmentID != nil && *req.DepartmentID != "" {
		id, err := uuid.Parse(*req.DepartmentID)
		if err != nil {
			return Period{}, errors.New("invalid department_id")
		}
		p.DepartmentID = &id
	}
	created, err := s.repo.CreatePeriod(ctx, p)
	if err != nil {
		return Period{}, err
	}

	deptIDs, err := parseUUIDs(req.DepartmentIDs)
	if err != nil {
		return Period{}, errors.New("invalid department_ids")
	}
	if created.DepartmentID != nil {
		deptIDs = appendUnique(deptIDs, *created.DepartmentID)
	}
	sectionIDs, err := parseUUIDs(req.SectionIDs)
	if err != nil {
		return Period{}, errors.New("invalid section_ids")
	}
	if err := s.repo.SetPeriodTargets(ctx, created.ID, deptIDs, sectionIDs); err != nil {
		return Period{}, err
	}
	if len(req.Criteria) > 0 {
		if err := s.repo.SetCriteria(ctx, created.ID, req.Criteria); err != nil {
			return Period{}, err
		}
	}
	created.DepartmentIDs = deptIDs
	created.SectionIDs = sectionIDs
	return created, nil
}

func parseUUIDs(in []string) ([]uuid.UUID, error) {
	out := make([]uuid.UUID, 0, len(in))
	for _, s := range in {
		if s == "" {
			continue
		}
		id, err := uuid.Parse(s)
		if err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, nil
}

func appendUnique(ids []uuid.UUID, id uuid.UUID) []uuid.UUID {
	for _, x := range ids {
		if x == id {
			return ids
		}
	}
	return append(ids, id)
}

func (s *Service) GetPeriodWithScores(ctx context.Context, id uuid.UUID) (Period, []Score, error) {
	p, err := s.repo.GetPeriod(ctx, id)
	if err != nil {
		return Period{}, nil, ErrNotFound
	}
	scores, err := s.repo.ListScores(ctx, id)
	if err != nil {
		return Period{}, nil, err
	}
	return p, scores, nil
}

func (s *Service) CreateCompetency(ctx context.Context, req CreateCompetencyRequest) (Competency, error) {
	code, err := s.uniqueCompetencyCode(ctx, deriveCompetencyCode(req.Name, req.Kind))
	if err != nil {
		return Competency{}, err
	}
	maxOrder, err := s.repo.MaxCompetencySortOrder(ctx)
	if err != nil {
		return Competency{}, err
	}
	return s.repo.CreateCompetency(ctx, code, req, maxOrder+10)
}

func (s *Service) uniqueCompetencyCode(ctx context.Context, base string) (string, error) {
	if base == "" {
		base = "COMP"
	}
	candidate := base
	for i := 2; ; i++ {
		exists, err := s.repo.CompetencyCodeExists(ctx, candidate)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s%d", base, i)
		if i > 999 {
			return "", errors.New("could not generate unique competency code")
		}
	}
}

func (s *Service) ReorderCompetencies(ctx context.Context, ids []string) error {
	uuids := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		u, err := uuid.Parse(id)
		if err != nil {
			return fmt.Errorf("invalid id %q: %w", id, err)
		}
		uuids = append(uuids, u)
	}
	return s.repo.ReorderCompetencies(ctx, uuids)
}

func (s *Service) UpdateCompetency(ctx context.Context, id uuid.UUID, req UpdateCompetencyRequest) (Competency, error) {
	return s.repo.UpdateCompetency(ctx, id, req)
}

func (s *Service) DeleteCompetency(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteCompetency(ctx, id)
}

func (s *Service) UpsertScore(ctx context.Context, periodID uuid.UUID, req UpsertScoreRequest, principalID uuid.UUID) (Score, error) {
	if _, err := s.repo.GetPeriod(ctx, periodID); err != nil {
		return Score{}, ErrNotFound
	}
	roles, err := s.repo.MyRolesIn(ctx, periodID, principalID)
	if err != nil {
		return Score{}, err
	}
	if len(roles) == 0 {
		return Score{}, ErrNotParticipant
	}
	role := req.AssessorRole
	if role == "" {
		if len(roles) != 1 {
			return Score{}, errors.New("assessor_role required when participant has multiple roles")
		}
		role = roles[0]
	}
	found := false
	for _, r := range roles {
		if r == role {
			found = true
			break
		}
	}
	if !found {
		return Score{}, ErrRoleNotInParticipant
	}
	autoInterp := s.computeAutoInterp(ctx, req.EmployeeID, req.CompetencyID, req.Score)
	score, err := s.repo.UpsertScoreFor(ctx, periodID, req.EmployeeID, req.CompetencyID, principalID, role, req.Score, req.Feedback, autoInterp)
	if err != nil {
		return Score{}, err
	}
	if err := s.repo.MaybeFinalize(ctx, periodID, req.EmployeeID, req.CompetencyID); err != nil {
		return Score{}, err
	}
	return score, nil
}

// computeAutoInterp resolves the system-suggested interpretation text for a
// saved score (FR-AS7.2). Returns nil when no score or no configured text.
func (s *Service) computeAutoInterp(ctx context.Context, employeeID, competencyID uuid.UUID, score *float64) *string {
	if score == nil {
		return nil
	}
	lk, err := s.LookupInterpretationForScore(ctx, employeeID, competencyID, int(*score))
	if err != nil || !lk.Found {
		return nil
	}
	text := lk.Text
	return &text
}
