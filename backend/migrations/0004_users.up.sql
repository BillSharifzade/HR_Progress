-- Users, roles, history, refresh tokens, audit log.

CREATE TABLE users (
    id                     uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    username               text NOT NULL,
    email                  text,
    password_hash          text NOT NULL,
    full_name              text NOT NULL,
    department_id          uuid REFERENCES departments(id),
    section_id             uuid REFERENCES sections(id),
    grade_id               uuid REFERENCES grades(id),
    specialization         text,
    telegram_id            bigint,
    hired_at               date,
    must_change_password   boolean NOT NULL DEFAULT false,
    is_active              boolean NOT NULL DEFAULT true,
    deleted_at             timestamptz,
    created_at             timestamptz NOT NULL DEFAULT now(),
    updated_at             timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX users_username_uniq ON users (lower(username)) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX users_email_uniq ON users (lower(email)) WHERE deleted_at IS NULL AND email IS NOT NULL;
CREATE INDEX users_department_id_idx ON users (department_id);
CREATE INDEX users_section_id_idx ON users (section_id);
CREATE INDEX users_grade_id_idx ON users (grade_id);

CREATE TRIGGER users_set_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE user_roles (
    id                    uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id               uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role                  role_kind NOT NULL,
    scope_department_id   uuid REFERENCES departments(id),
    granted_by            uuid REFERENCES users(id),
    granted_at            timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX user_roles_uniq ON user_roles (user_id, role, COALESCE(scope_department_id, '00000000-0000-0000-0000-000000000000'::uuid));
CREATE INDEX user_roles_user_id_idx ON user_roles (user_id);

CREATE TABLE user_history (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_kind    history_event_kind NOT NULL,
    event_date    date NOT NULL,
    title         text NOT NULL,
    description   text,
    meta          jsonb,
    created_by    uuid REFERENCES users(id),
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX user_history_user_date_idx ON user_history (user_id, event_date DESC);

CREATE TABLE refresh_tokens (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash   text NOT NULL UNIQUE,
    issued_at    timestamptz NOT NULL DEFAULT now(),
    expires_at   timestamptz NOT NULL,
    revoked_at   timestamptz,
    user_agent   text,
    ip           inet
);

CREATE INDEX refresh_tokens_user_id_idx ON refresh_tokens (user_id);

CREATE TABLE audit_logs (
    id                    uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id              uuid REFERENCES users(id),
    action                audit_action NOT NULL,
    entity_type           text NOT NULL,
    entity_id             uuid,
    before_data           jsonb,
    after_data            jsonb,
    department_scope_id   uuid REFERENCES departments(id),
    ip                    inet,
    user_agent            text,
    created_at            timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX audit_logs_created_at_idx ON audit_logs (created_at DESC);
CREATE INDEX audit_logs_entity_idx ON audit_logs (entity_type, entity_id, created_at DESC);
CREATE INDEX audit_logs_dept_scope_idx ON audit_logs (department_scope_id, created_at DESC);
