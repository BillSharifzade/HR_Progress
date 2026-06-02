-- Allow user_roles to be scoped to a section as well as a department.
-- SECTION_HEAD grants require a section; DEPT_HEAD grants require a department.

ALTER TABLE user_roles ADD COLUMN IF NOT EXISTS scope_section_id uuid REFERENCES sections(id);

-- Replace the previous uniqueness index so the same role can be held for
-- different (department, section) scopes.
DROP INDEX IF EXISTS user_roles_uniq;

CREATE UNIQUE INDEX user_roles_uniq ON user_roles (
    user_id,
    role,
    COALESCE(scope_department_id, '00000000-0000-0000-0000-000000000000'::uuid),
    COALESCE(scope_section_id,    '00000000-0000-0000-0000-000000000000'::uuid)
);

CREATE INDEX IF NOT EXISTS user_roles_scope_section_idx ON user_roles (scope_section_id);
