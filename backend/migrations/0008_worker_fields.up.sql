-- Positions reference table, worker profile fields, certifications, new role values.

-- Extend role enum before any table DDL that might reference it.
ALTER TYPE role_kind ADD VALUE IF NOT EXISTS 'SECTION_HEAD';
ALTER TYPE role_kind ADD VALUE IF NOT EXISTS 'ATS';
ALTER TYPE role_kind ADD VALUE IF NOT EXISTS 'BOOK_SPACE';

-- Job-title reference (справочник должностей).
CREATE TABLE positions (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name        text NOT NULL,
    is_active   boolean NOT NULL DEFAULT true,
    deleted_at  timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX positions_name_uniq ON positions (lower(name)) WHERE deleted_at IS NULL;
CREATE TRIGGER positions_set_updated_at BEFORE UPDATE ON positions
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Worker profile additions.
ALTER TABLE users
    ADD COLUMN personnel_number text,
    ADD COLUMN birth_date       date,
    ADD COLUMN position_id      uuid REFERENCES positions(id),
    ADD COLUMN hobbies          text;

CREATE UNIQUE INDEX users_personnel_number_uniq ON users (personnel_number)
    WHERE deleted_at IS NULL AND personnel_number IS NOT NULL;
CREATE INDEX users_position_id_idx ON users (position_id);

-- Certifications (free-text list per worker).
CREATE TABLE user_certifications (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title       text NOT NULL,
    issued_by   text,
    issued_at   date,
    expires_at  date,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX user_certifications_user_id_idx ON user_certifications (user_id);
CREATE TRIGGER user_certifications_set_updated_at BEFORE UPDATE ON user_certifications
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
