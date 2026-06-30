package competency

import (
	"time"

	"github.com/google/uuid"
)

type Kind string

const (
	KindLK Kind = "LK"
	KindUK Kind = "UK"
	KindPK Kind = "PK"
)

type Competency struct {
	ID           uuid.UUID `json:"id"`
	Code         string    `json:"code"`
	Kind         Kind      `json:"kind"`
	Name         string    `json:"name"`
	Description  *string   `json:"description,omitempty"`
	WhyImportant *string   `json:"why_important,omitempty"`
	SortOrder    int       `json:"sort_order"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Requirement struct {
	ID             uuid.UUID `json:"id"`
	DepartmentID   uuid.UUID `json:"department_id"`
	CompetencyID   uuid.UUID `json:"competency_id"`
	CompetencyCode string    `json:"competency_code,omitempty"`
	CompetencyName string    `json:"competency_name,omitempty"`
	CompetencyKind Kind      `json:"competency_kind,omitempty"`
	GradeID        uuid.UUID `json:"grade_id"`
	GradeLevel     int       `json:"grade_level,omitempty"`
	GradeName      string    `json:"grade_name,omitempty"`
	RequiredMin    *int      `json:"required_min"`
	IsKey          bool      `json:"is_key"`
	Description    *string   `json:"description,omitempty"`
}

type UpsertRequirementRequest struct {
	CompetencyID uuid.UUID `json:"competency_id" validate:"required"`
	GradeID      uuid.UUID `json:"grade_id"      validate:"required"`
	RequiredMin  *int      `json:"required_min"`
	IsKey        bool      `json:"is_key"`
	Description  *string   `json:"description"`
}

type Period struct {
	ID            uuid.UUID   `json:"id"`
	Title         string      `json:"title"`
	DepartmentID  *uuid.UUID  `json:"department_id,omitempty"`
	PeriodStart   time.Time   `json:"period_start"`
	PeriodEnd     time.Time   `json:"period_end"`
	IsActive      bool        `json:"is_active"`
	Status        string      `json:"status"`
	GroupSize     int         `json:"group_size"`
	ConfirmedAt   *time.Time  `json:"confirmed_at,omitempty"`
	PublishedAt   *time.Time  `json:"published_at,omitempty"`
	CreatedBy     *uuid.UUID  `json:"created_by,omitempty"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
	DepartmentIDs []uuid.UUID `json:"department_ids,omitempty"`
	SectionIDs    []uuid.UUID `json:"section_ids,omitempty"`
}

type CreatePeriodRequest struct {
	Title         string           `json:"title"        validate:"required,min=1,max=200"`
	DepartmentID  *string          `json:"department_id"`
	PeriodStart   string           `json:"period_start" validate:"required"`
	PeriodEnd     string           `json:"period_end"   validate:"required"`
	GroupSize     *int             `json:"group_size"   validate:"omitempty,min=1"`
	DepartmentIDs []string         `json:"department_ids"`
	SectionIDs    []string         `json:"section_ids"`
	Criteria      []CriterionInput `json:"criteria"`
}

// ── Criteria (FR-AS3) ────────────────────────────────────────────────────────

// Criterion is a competency selected for a campaign with a passing score.
type Criterion struct {
	ID             uuid.UUID `json:"id"`
	PeriodID       uuid.UUID `json:"period_id"`
	CompetencyID   uuid.UUID `json:"competency_id"`
	CompetencyCode string    `json:"competency_code,omitempty"`
	Name           string    `json:"name"`
	Description    *string   `json:"description,omitempty"`
	MinScore       *int      `json:"min_score,omitempty"`
	SortOrder      int       `json:"sort_order"`
}

type CriterionInput struct {
	CompetencyID uuid.UUID `json:"competency_id" validate:"required"`
	Name         *string   `json:"name"`
	Description  *string   `json:"description"`
	MinScore     *int      `json:"min_score" validate:"omitempty,min=1,max=10"`
}

type SetCriteriaRequest struct {
	Criteria []CriterionInput `json:"criteria" validate:"required,min=1,dive"`
}

// ── Assessees (FR-AS2) ───────────────────────────────────────────────────────

// Assessee is an employee being evaluated in a campaign.
type Assessee struct {
	ID           uuid.UUID  `json:"id"`
	PeriodID     uuid.UUID  `json:"period_id"`
	UserID       uuid.UUID  `json:"user_id"`
	FullName     string     `json:"full_name,omitempty"`
	Status       string     `json:"status"`
	GradeID      *uuid.UUID `json:"grade_id,omitempty"`
	GradeName    *string    `json:"grade_name,omitempty"`
	DepartmentID *uuid.UUID `json:"department_id,omitempty"`
	AddedAt      time.Time  `json:"added_at"`
}

// AddAssesseesRequest assigns assessees by department, section, and/or individuals.
type AddAssesseesRequest struct {
	UserIDs       []string `json:"user_ids"`
	DepartmentIDs []string `json:"department_ids"`
	SectionIDs    []string `json:"section_ids"`
}

// ── Per-assessee assessors (FR-AS4) ──────────────────────────────────────────

type AssesseeAssessor struct {
	ID             uuid.UUID `json:"id"`
	PeriodID       uuid.UUID `json:"period_id"`
	AssesseeUserID uuid.UUID `json:"assessee_user_id"`
	AssessorUserID uuid.UUID `json:"assessor_user_id"`
	AssessorName   string    `json:"assessor_name,omitempty"`
}

type SetAssessorsRequest struct {
	AssessorUserIDs []string `json:"assessor_user_ids" validate:"required"`
}

// ── Interpretation reference / справочник (FR-AS7.2) ─────────────────────────

type Interpretation struct {
	ID             uuid.UUID  `json:"id"`
	DepartmentID   uuid.UUID  `json:"department_id"`
	DepartmentName string     `json:"department_name,omitempty"`
	GradeID        uuid.UUID  `json:"grade_id"`
	GradeName      string     `json:"grade_name,omitempty"`
	CompetencyID   uuid.UUID  `json:"competency_id"`
	CompetencyName string     `json:"competency_name,omitempty"`
	Score          int        `json:"score"`
	Text           string     `json:"text"`
	Version        int        `json:"version"`
	IsActive       bool       `json:"is_active"`
	CreatedBy      *uuid.UUID `json:"created_by,omitempty"`
	UpdatedBy      *uuid.UUID `json:"updated_by,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type UpsertInterpretationRequest struct {
	DepartmentID uuid.UUID `json:"department_id" validate:"required"`
	GradeID      uuid.UUID `json:"grade_id"      validate:"required"`
	CompetencyID uuid.UUID `json:"competency_id" validate:"required"`
	Score        int       `json:"score"         validate:"required,min=1,max=10"`
	Text         string    `json:"text"          validate:"required,min=1"`
}

// CopyInterpretationsRequest copies all active interpretations from one
// (department, grade) to another (FR-AS7.2.4).
type CopyInterpretationsRequest struct {
	FromDepartmentID uuid.UUID  `json:"from_department_id" validate:"required"`
	ToDepartmentID   uuid.UUID  `json:"to_department_id"   validate:"required"`
	FromGradeID      *uuid.UUID `json:"from_grade_id"`
	ToGradeID        *uuid.UUID `json:"to_grade_id"`
	Overwrite        bool       `json:"overwrite"`
}

type InterpretationHistoryEntry struct {
	ID               uuid.UUID  `json:"id"`
	InterpretationID *uuid.UUID `json:"interpretation_id,omitempty"`
	Score            int        `json:"score"`
	Text             string     `json:"text"`
	Version          int        `json:"version"`
	Action           string     `json:"action"`
	ChangedBy        *uuid.UUID `json:"changed_by,omitempty"`
	ChangedAt        time.Time  `json:"changed_at"`
}

// InterpretationLookup is the response for the auto-interpretation lookup.
type InterpretationLookup struct {
	Found bool   `json:"found"`
	Text  string `json:"text,omitempty"`
}

// ── Learning groups (FR-AS13) ────────────────────────────────────────────────

type LearningGroup struct {
	ID                   uuid.UUID     `json:"id"`
	PeriodID             uuid.UUID     `json:"period_id"`
	GroupNo              int           `json:"group_no"`
	ScoreMin             *float64      `json:"score_min,omitempty"`
	ScoreMax             *float64      `json:"score_max,omitempty"`
	StrengthCompetencyID *uuid.UUID    `json:"strength_competency_id,omitempty"`
	StrengthName         string        `json:"strength_name,omitempty"`
	StrengthScore        *float64      `json:"strength_score,omitempty"`
	Confirmed            bool          `json:"confirmed"`
	FormedAt             time.Time     `json:"formed_at"`
	Members              []GroupMember `json:"members"`
	DevZones             []DevZone     `json:"dev_zones"`
}

type GroupMember struct {
	ID       uuid.UUID `json:"id"`
	GroupID  uuid.UUID `json:"group_id"`
	UserID   uuid.UUID `json:"user_id"`
	FullName string    `json:"full_name,omitempty"`
	AvgScore float64   `json:"avg_score"`
	Position int       `json:"position"`
}

type DevZone struct {
	CompetencyID   uuid.UUID `json:"competency_id"`
	CompetencyName string    `json:"competency_name,omitempty"`
	AvgScore       float64   `json:"avg_score"`
	Rank           int       `json:"rank"`
}

type MoveMemberRequest struct {
	UserID    uuid.UUID `json:"user_id"        validate:"required"`
	ToGroupID uuid.UUID `json:"to_group_id"    validate:"required"`
}

type SetGroupSizeRequest struct {
	GroupSize int `json:"group_size" validate:"required,min=1"`
}

type GroupJournalEntry struct {
	ID      uuid.UUID  `json:"id"`
	GroupID *uuid.UUID `json:"group_id,omitempty"`
	Action  string     `json:"action"`
	Detail  *string    `json:"detail,omitempty"`
	ActorID *uuid.UUID `json:"actor_id,omitempty"`
	At      time.Time  `json:"at"`
}

type Score struct {
	ID                 uuid.UUID  `json:"id"`
	PeriodID           uuid.UUID  `json:"period_id"`
	EmployeeID         uuid.UUID  `json:"employee_id"`
	CompetencyID       uuid.UUID  `json:"competency_id"`
	AssessorRole       string     `json:"assessor_role"`
	Score              *float64   `json:"score"`
	Feedback           *string    `json:"feedback,omitempty"`
	AutoInterpretation *string    `json:"auto_interpretation,omitempty"`
	AssessedBy         *uuid.UUID `json:"assessed_by,omitempty"`
	AssessedAt         *time.Time `json:"assessed_at,omitempty"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type UpsertScoreRequest struct {
	EmployeeID   uuid.UUID `json:"employee_id"   validate:"required"`
	CompetencyID uuid.UUID `json:"competency_id" validate:"required"`
	AssessorRole string    `json:"assessor_role" validate:"omitempty,oneof=HEAD DEPT_HEAD HRA DCR_HEAD ASSESSOR"`
	Score        *float64  `json:"score" validate:"omitempty,min=1,max=10"`
	Feedback     *string   `json:"feedback"`
}

// Participant is one user assigned to score in a period.
type Participant struct {
	ID       uuid.UUID `json:"id"`
	PeriodID uuid.UUID `json:"period_id"`
	UserID   uuid.UUID `json:"user_id"`
	Role     string    `json:"role"`
	UserName string    `json:"user_name,omitempty"`
	FullName string    `json:"full_name,omitempty"`
	AddedAt  time.Time `json:"added_at"`
}

type ParticipantInput struct {
	UserID uuid.UUID `json:"user_id" validate:"required"`
	Role   string    `json:"role"    validate:"required,oneof=HEAD DEPT_HEAD HRA DCR_HEAD ASSESSOR"`
}

type AddParticipantsRequest struct {
	Participants []ParticipantInput `json:"participants" validate:"required,min=1,dive"`
}

// ConsolidatedScore is the finalized matrix mark for (period, worker,
// competency): the average of equally-weighted voices — the assessor group as
// one voice (their average) plus each HEAD / DEPT_HEAD / DCR_HEAD (see
// MaybeFinalize).
type ConsolidatedScore struct {
	ID           uuid.UUID `json:"id"`
	PeriodID     uuid.UUID `json:"period_id"`
	EmployeeID   uuid.UUID `json:"employee_id"`
	CompetencyID uuid.UUID `json:"competency_id"`
	AvgScore     float64   `json:"avg_score"`
	FinalizedAt  time.Time `json:"finalized_at"`
}

// MyPeriod is the listing of a period a user is participating in.
type MyPeriod struct {
	PeriodID     uuid.UUID  `json:"period_id"`
	Title        string     `json:"title"`
	PeriodStart  time.Time  `json:"period_start"`
	PeriodEnd    time.Time  `json:"period_end"`
	IsActive     bool       `json:"is_active"`
	Roles        []string   `json:"roles"`
	Department   *string    `json:"department,omitempty"`
	DepartmentID *uuid.UUID `json:"department_id,omitempty"`
}

type Department struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	IsActive    bool   `json:"is_active"`
}

// MatrixRow is a single competency row in the assessment matrix view.
type MatrixRow struct {
	Competency   Competency      `json:"competency"`
	Requirements map[string]int  `json:"requirements"` // grade_level → required_min
	KeyGrades    map[string]bool `json:"key_grades"`   // grade_level → is_key
	Scores       []Score         `json:"scores"`
}

type Employee struct {
	ID         uuid.UUID  `json:"id"`
	FullName   string     `json:"full_name"`
	GradeID    *uuid.UUID `json:"grade_id,omitempty"`
	GradeName  *string    `json:"grade_name,omitempty"`
	GradeLevel *int       `json:"grade_level,omitempty"`
	SectionID  *uuid.UUID `json:"section_id,omitempty"`
}

type Grade struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Level       int       `json:"level"`
	Description *string   `json:"description,omitempty"`
	IsActive    bool      `json:"is_active"`
}

type CreateDepartmentRequest struct {
	Name        string  `json:"name"        validate:"required,min=1,max=200"`
	Description *string `json:"description"`
}

type UpdateDepartmentRequest struct {
	Name        string  `json:"name"        validate:"required,min=1,max=200"`
	Description *string `json:"description"`
	IsActive    bool    `json:"is_active"`
}

type CreateCompetencyRequest struct {
	Kind         Kind    `json:"kind"          validate:"required,oneof=LK UK PK"`
	Name         string  `json:"name"          validate:"required,min=1,max=200"`
	Description  *string `json:"description"`
	WhyImportant *string `json:"why_important"`
}

type UpdateCompetencyRequest struct {
	Kind         Kind    `json:"kind"          validate:"required,oneof=LK UK PK"`
	Name         string  `json:"name"          validate:"required,min=1,max=200"`
	Description  *string `json:"description"`
	WhyImportant *string `json:"why_important"`
	IsActive     bool    `json:"is_active"`
}

type ReorderCompetenciesRequest struct {
	IDs []string `json:"ids" validate:"required,min=1"`
}
