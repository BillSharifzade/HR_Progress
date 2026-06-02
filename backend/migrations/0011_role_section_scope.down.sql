DROP INDEX IF EXISTS user_roles_scope_section_idx;
DROP INDEX IF EXISTS user_roles_uniq;

CREATE UNIQUE INDEX user_roles_uniq ON user_roles (
    user_id,
    role,
    COALESCE(scope_department_id, '00000000-0000-0000-0000-000000000000'::uuid)
);

ALTER TABLE user_roles DROP COLUMN IF EXISTS scope_section_id;
