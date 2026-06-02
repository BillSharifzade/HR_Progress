package rbac

import (
	"testing"

	"github.com/google/uuid"
)

func TestAllow(t *testing.T) {
	dept1 := uuid.New()
	dept2 := uuid.New()

	cases := []struct {
		name   string
		actor  Actor
		action Action
		target Target
		want   bool
	}{
		{
			name:   "hr admin can do anything",
			actor:  Actor{Roles: []string{"HR_ADMIN"}},
			action: ActionRolesGrant,
			target: Target{},
			want:   true,
		},
		{
			name:   "dept head can edit users in own dept",
			actor:  Actor{Roles: []string{"DEPT_HEAD"}, ScopeDepartmentIDs: []uuid.UUID{dept1}},
			action: ActionUsersEdit,
			target: Target{DepartmentID: &dept1},
			want:   true,
		},
		{
			name:   "dept head cannot edit users in other dept",
			actor:  Actor{Roles: []string{"DEPT_HEAD"}, ScopeDepartmentIDs: []uuid.UUID{dept1}},
			action: ActionUsersEdit,
			target: Target{DepartmentID: &dept2},
			want:   false,
		},
		{
			name:   "dept head cannot grant roles",
			actor:  Actor{Roles: []string{"DEPT_HEAD"}, ScopeDepartmentIDs: []uuid.UUID{dept1}},
			action: ActionRolesGrant,
			target: Target{DepartmentID: &dept1},
			want:   false,
		},
		{
			name:   "assessor alone cannot edit users",
			actor:  Actor{Roles: []string{"ASSESSOR"}},
			action: ActionUsersEdit,
			target: Target{DepartmentID: &dept1},
			want:   false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Allow(tc.actor, tc.action, tc.target)
			if got != tc.want {
				t.Fatalf("Allow(%v, %s, %+v) = %v, want %v", tc.actor.Roles, tc.action, tc.target, got, tc.want)
			}
		})
	}
}
