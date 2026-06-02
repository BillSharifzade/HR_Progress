package auth

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID                 uuid.UUID   `json:"id"`
	PersonnelNumber    *string     `json:"personnel_number,omitempty"`
	Username           string      `json:"username"`
	Email              *string     `json:"email,omitempty"`
	FullName           string      `json:"full_name"`
	BirthDate          *time.Time  `json:"birth_date,omitempty"`
	DepartmentID       *uuid.UUID  `json:"department_id,omitempty"`
	SectionID          *uuid.UUID  `json:"section_id,omitempty"`
	GradeID            *uuid.UUID  `json:"grade_id,omitempty"`
	PositionID         *uuid.UUID  `json:"position_id,omitempty"`
	PositionName       *string     `json:"position_name,omitempty"`
	Specialization     *string     `json:"specialization,omitempty"`
	TelegramID         *int64      `json:"telegram_id,omitempty"`
	HiredAt            *time.Time  `json:"hired_at,omitempty"`
	Hobbies            *string     `json:"hobbies,omitempty"`
	MustChangePassword bool        `json:"must_change_password"`
	IsActive           bool        `json:"is_active"`
	Roles              []string    `json:"roles"`
	ScopeDepartmentIDs []uuid.UUID `json:"scope_department_ids,omitempty"`
	ScopeSectionIDs    []uuid.UUID `json:"scope_section_ids,omitempty"`
}


type LoginRequest struct {
	Username string `json:"username" validate:"required,min=1,max=100"`
	Password string `json:"password" validate:"required,min=1,max=200"`
}

type LoginResponse struct {
	AccessToken string    `json:"access_token"`
	ExpiresAt   time.Time `json:"expires_at"`
	User        User      `json:"user"`
}

