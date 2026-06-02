package workers

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Worker struct {
	ID              uuid.UUID  `json:"id"`
	Username        string     `json:"username"`
	EmployeeNo      int64      `json:"employee_no"`
	PersonnelNumber *string    `json:"personnel_number,omitempty"`
	OneFUserID      *int64     `json:"one_f_user_id,omitempty"`
	OneFIsManager   bool       `json:"one_f_is_manager"`
	PhoneNumber     *string    `json:"phone_number,omitempty"`
	LastSyncedAt    *time.Time `json:"last_synced_at,omitempty"`
	FullName        string     `json:"full_name"`
	Email           *string    `json:"email,omitempty"`
	BirthDate       *time.Time `json:"birth_date,omitempty"`
	DepartmentID    *uuid.UUID `json:"department_id,omitempty"`
	DepartmentName  *string    `json:"department_name,omitempty"`
	SectionID       *uuid.UUID `json:"section_id,omitempty"`
	SectionName     *string    `json:"section_name,omitempty"`
	GradeID         *uuid.UUID `json:"grade_id,omitempty"`
	GradeName       *string    `json:"grade_name,omitempty"`
	GradeLevel      *int       `json:"grade_level,omitempty"`
	PositionID      *uuid.UUID `json:"position_id,omitempty"`
	PositionName    *string    `json:"position_name,omitempty"`
	Position        *string    `json:"position,omitempty"`
	Specialization  *string    `json:"specialization,omitempty"`
	TelegramID      *int64     `json:"telegram_id,omitempty"`
	HiredAt         *time.Time `json:"hired_at,omitempty"`
	Hobbies         *string    `json:"hobbies,omitempty"`
	IsActive        bool       `json:"is_active"`
	Roles           []string   `json:"roles"`
}

type WorkerSummary struct {
	ID              uuid.UUID  `json:"id"`
	EmployeeNo      int64      `json:"employee_no"`
	PersonnelNumber *string    `json:"personnel_number,omitempty"`
	OneFUserID      *int64     `json:"one_f_user_id,omitempty"`
	FullName        string     `json:"full_name"`
	DepartmentName  *string    `json:"department_name,omitempty"`
	SectionName     *string    `json:"section_name,omitempty"`
	GradeName       *string    `json:"grade_name,omitempty"`
	GradeLevel      *int       `json:"grade_level,omitempty"`
	PositionName    *string    `json:"position_name,omitempty"`
	HiredAt         *time.Time `json:"hired_at,omitempty"`
	IsActive        bool       `json:"is_active"`
}

type History struct {
	ID          uuid.UUID       `json:"id"`
	UserID      uuid.UUID       `json:"user_id"`
	EventKind   string          `json:"event_kind"`
	EventDate   time.Time       `json:"event_date"`
	Title       string          `json:"title"`
	Description *string         `json:"description,omitempty"`
	Meta        json.RawMessage `json:"meta,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
}

type Certification struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"user_id"`
	Title     string     `json:"title"`
	IssuedBy  *string    `json:"issued_by,omitempty"`
	IssuedAt  *time.Time `json:"issued_at,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type Position struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	IsActive bool      `json:"is_active"`
}

type ListFilter struct {
	DepartmentID    *uuid.UUID
	SectionID       *uuid.UUID
	GradeID         *uuid.UUID
	Search          string
	IncludeInactive bool
}

type CreateHistoryRequest struct {
	EventKind   string          `json:"event_kind"   validate:"required,oneof=HIRED PROMOTED TRANSFERRED EXTERNAL_EXPERIENCE COMMENT OTHER"`
	EventDate   string          `json:"event_date"   validate:"required"`
	Title       string          `json:"title"        validate:"required,min=1,max=500"`
	Description *string         `json:"description"`
	Meta        json.RawMessage `json:"meta"`
}

type UpsertCertificationRequest struct {
	Title     string  `json:"title"      validate:"required,min=1,max=300"`
	IssuedBy  *string `json:"issued_by"`
	IssuedAt  *string `json:"issued_at"`
	ExpiresAt *string `json:"expires_at"`
}

type CreatePositionRequest struct {
	Name string `json:"name" validate:"required,min=1,max=200"`
}

type Section struct {
	ID           uuid.UUID `json:"id"`
	DepartmentID uuid.UUID `json:"department_id"`
	Code         *string   `json:"code,omitempty"`
	Name         string    `json:"name"`
	Description  *string   `json:"description,omitempty"`
	IsActive     bool      `json:"is_active"`
}

type CreateSectionRequest struct {
	DepartmentID uuid.UUID
	Code         *string
	Name         string
	Description  *string
}

type UpdateSectionRequest struct {
	Name        string
	Description *string
	IsActive    bool
}

type CreateWorkerRequest struct {
	Username        string     `json:"username"         validate:"omitempty,min=2,max=100"`
	FullName        string     `json:"full_name"        validate:"required,min=1,max=255"`
	Email           *string    `json:"email"`
	PersonnelNumber *string    `json:"personnel_number"`
	BirthDate       *string    `json:"birth_date"`
	DepartmentID    *uuid.UUID `json:"department_id"`
	SectionID       *uuid.UUID `json:"section_id"`
	GradeID         *uuid.UUID `json:"grade_id"`
	Position        *string    `json:"position"`
	Specialization  *string    `json:"specialization"`
	TelegramID      *int64     `json:"telegram_id"`
	HiredAt         *string    `json:"hired_at"`
	Hobbies         *string    `json:"hobbies"`
}

type RoleAssignment struct {
	ID                uuid.UUID  `json:"id"`
	UserID            uuid.UUID  `json:"user_id"`
	Role              string     `json:"role"`
	ScopeDepartmentID *uuid.UUID `json:"scope_department_id,omitempty"`
	ScopeDepartment   *string    `json:"scope_department,omitempty"`
	ScopeSectionID    *uuid.UUID `json:"scope_section_id,omitempty"`
	ScopeSection      *string    `json:"scope_section,omitempty"`
	GrantedByID       *uuid.UUID `json:"granted_by_id,omitempty"`
	GrantedByName     *string    `json:"granted_by_name,omitempty"`
	GrantedAt         time.Time  `json:"granted_at"`
}

type GrantRoleRequest struct {
	Role              string     `json:"role"                validate:"required,oneof=HR_ADMIN DEPT_HEAD SECTION_HEAD ASSESSOR PRECEPTOR ATS BOOK_SPACE"`
	ScopeDepartmentID *uuid.UUID `json:"scope_department_id"`
	ScopeSectionID    *uuid.UUID `json:"scope_section_id"`
}

type UpdateWorkerRequest struct {
	FullName        string     `json:"full_name"        validate:"required,min=1,max=255"`
	Email           *string    `json:"email"`
	PersonnelNumber *string    `json:"personnel_number"`
	BirthDate       *string    `json:"birth_date"`
	DepartmentID    *uuid.UUID `json:"department_id"`
	SectionID       *uuid.UUID `json:"section_id"`
	GradeID         *uuid.UUID `json:"grade_id"`
	Position        *string    `json:"position"`
	Specialization  *string    `json:"specialization"`
	TelegramID      *int64     `json:"telegram_id"`
	HiredAt         *string    `json:"hired_at"`
	Hobbies         *string    `json:"hobbies"`
}
