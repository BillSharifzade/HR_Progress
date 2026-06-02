package rbac

import (
	"github.com/google/uuid"
)

type Action string

const (
	ActionUsersCreate Action = "users.create"
	ActionUsersEdit   Action = "users.edit"
	ActionRolesGrant  Action = "roles.grant"
	ActionAuditView   Action = "audit.view"
	// Phase 2+ actions will be added here as new resources land.
)

type Actor struct {
	UserID             uuid.UUID
	Roles              []string
	ScopeDepartmentIDs []uuid.UUID
}

func (a Actor) hasRole(role string) bool {
	for _, r := range a.Roles {
		if r == role {
			return true
		}
	}
	return false
}

func (a Actor) headsDepartment(deptID uuid.UUID) bool {
	for _, id := range a.ScopeDepartmentIDs {
		if id == deptID {
			return true
		}
	}
	return false
}

type Target struct {
	DepartmentID *uuid.UUID
}

// Allow returns true if the actor may perform the action on the target.
func Allow(actor Actor, action Action, target Target) bool {
	if actor.hasRole("HR_ADMIN") {
		return true
	}
	switch action {
	case ActionUsersCreate, ActionUsersEdit, ActionAuditView:
		if actor.hasRole("DEPT_HEAD") && target.DepartmentID != nil && actor.headsDepartment(*target.DepartmentID) {
			return true
		}
	case ActionRolesGrant:
		// Only HR_ADMIN can grant roles.
		return false
	}
	return false
}
