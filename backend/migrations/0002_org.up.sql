-- Departments and sections (отделы).

CREATE TABLE departments (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    code         text NOT NULL,
    name         text NOT NULL,
    description  text,
    is_active    boolean NOT NULL DEFAULT true,
    deleted_at   timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX departments_code_uniq ON departments (code) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX departments_name_uniq ON departments (name) WHERE deleted_at IS NULL;

CREATE TRIGGER departments_set_updated_at BEFORE UPDATE ON departments
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE sections (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    department_id  uuid NOT NULL REFERENCES departments(id),
    name           text NOT NULL,
    description    text,
    is_active      boolean NOT NULL DEFAULT true,
    deleted_at     timestamptz,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX sections_dept_name_uniq ON sections (department_id, name) WHERE deleted_at IS NULL;
CREATE INDEX sections_department_id_idx ON sections (department_id);

CREATE TRIGGER sections_set_updated_at BEFORE UPDATE ON sections
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
