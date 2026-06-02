-- Competency matrix tables.

CREATE TYPE competency_kind AS ENUM ('LK', 'UK', 'PK');

CREATE TABLE competencies (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    code          text NOT NULL,
    kind          competency_kind NOT NULL,
    name          text NOT NULL,
    description   text,
    why_important text,
    sort_order    integer NOT NULL DEFAULT 0,
    is_active     boolean NOT NULL DEFAULT true,
    deleted_at    timestamptz,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX competencies_code_uniq ON competencies (code) WHERE deleted_at IS NULL;

CREATE TRIGGER competencies_set_updated_at BEFORE UPDATE ON competencies
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Per-department, per-grade minimum score requirements for each competency.
CREATE TABLE dept_competency_requirements (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    department_id   uuid NOT NULL REFERENCES departments(id),
    competency_id   uuid NOT NULL REFERENCES competencies(id),
    grade_id        uuid NOT NULL REFERENCES grades(id),
    required_min    integer,
    is_key          boolean NOT NULL DEFAULT false,
    description     text,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (department_id, competency_id, grade_id)
);

CREATE INDEX dcr_dept_grade_idx ON dept_competency_requirements (department_id, grade_id);

CREATE TRIGGER dcr_set_updated_at BEFORE UPDATE ON dept_competency_requirements
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Assessment cycles.
CREATE TABLE assessment_periods (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    title         text NOT NULL,
    department_id uuid REFERENCES departments(id),
    period_start  date NOT NULL,
    period_end    date NOT NULL,
    is_active     boolean NOT NULL DEFAULT true,
    created_by    uuid REFERENCES users(id),
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX ap_dept_idx ON assessment_periods (department_id);

CREATE TRIGGER ap_set_updated_at BEFORE UPDATE ON assessment_periods
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Individual competency scores per (period, employee, competency, assessor_role).
CREATE TABLE assessment_scores (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    period_id     uuid NOT NULL REFERENCES assessment_periods(id) ON DELETE CASCADE,
    employee_id   uuid NOT NULL REFERENCES users(id),
    competency_id uuid NOT NULL REFERENCES competencies(id),
    assessor_role text NOT NULL CHECK (assessor_role IN ('HEAD', 'DEPT_HEAD', 'HRA', 'DCR_HEAD')),
    score         numeric(4,1) CHECK (score >= 0 AND score <= 10),
    feedback      text,
    assessed_by   uuid REFERENCES users(id),
    assessed_at   timestamptz,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    UNIQUE (period_id, employee_id, competency_id, assessor_role)
);

CREATE INDEX as_period_employee_idx ON assessment_scores (period_id, employee_id);

CREATE TRIGGER as_set_updated_at BEFORE UPDATE ON assessment_scores
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
