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
	ID             uuid.UUID  `json:"id"`
	DepartmentID   uuid.UUID  `json:"department_id"`
	CompetencyID   uuid.UUID  `json:"competency_id"`
	CompetencyCode string     `json:"competency_code,omitempty"`
	CompetencyName string     `json:"competency_name,omitempty"`
	CompetencyKind Kind       `json:"competency_kind,omitempty"`
	GradeID        uuid.UUID  `json:"grade_id"`
	GradeLevel     int        `json:"grade_level,omitempty"`
	GradeName      string     `json:"grade_name,omitempty"`
	RequiredMin    *int       `json:"required_min"`
	IsKey          bool       `json:"is_key"`
	Description    *string    `json:"description,omitempty"`
}

type UpsertRequirementRequest struct {
	CompetencyID uuid.UUID `json:"competency_id" validate:"required"`
	GradeID      uuid.UUID `json:"grade_id"      validate:"required"`
	RequiredMin  *int      `json:"required_min"`
	IsKey        bool      `json:"is_key"`
	Description  *string   `json:"description"`
}

type Period struct {
	ID           uuid.UUID  `json:"id"`
	Title        string     `json:"title"`
	DepartmentID *uuid.UUID `json:"department_id,omitempty"`
	PeriodStart  time.Time  `json:"period_start"`
	PeriodEnd    time.Time  `json:"period_end"`
	IsActive     bool       `json:"is_active"`
	CreatedBy    *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type CreatePeriodRequest struct {
	Title        string    `json:"title"        validate:"required,min=1,max=200"`
	DepartmentID *string   `json:"department_id"`
	PeriodStart  string    `json:"period_start" validate:"required"`
	PeriodEnd    string    `json:"period_end"   validate:"required"`
}

type Score struct {
	ID           uuid.UUID  `json:"id"`
	PeriodID     uuid.UUID  `json:"period_id"`
	EmployeeID   uuid.UUID  `json:"employee_id"`
	CompetencyID uuid.UUID  `json:"competency_id"`
	AssessorRole string     `json:"assessor_role"`
	Score        *float64   `json:"score"`
	Feedback     *string    `json:"feedback,omitempty"`
	AssessedBy   *uuid.UUID `json:"assessed_by,omitempty"`
	AssessedAt   *time.Time `json:"assessed_at,omitempty"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type UpsertScoreRequest struct {
	EmployeeID   uuid.UUID `json:"employee_id"   validate:"required"`
	CompetencyID uuid.UUID `json:"competency_id" validate:"required"`
	AssessorRole string    `json:"assessor_role" validate:"omitempty,oneof=HEAD DEPT_HEAD HRA DCR_HEAD ASSESSOR"`
	Score        *float64  `json:"score"`
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

// ConsolidatedScore is the average ASSESSOR-group mark for (period, worker, competency).
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
