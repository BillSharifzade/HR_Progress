-- Full Assessment module schema (per desc.md ТЗ).
-- Covers: campaign lifecycle/statuses, multi-dept/section targeting, criteria,
-- assessees + per-campaign status, per-assessee assessor assignment,
-- text-interpretation reference (справочник) + history, learning-group engine,
-- and Talent Profile sync fields. Scale is enforced 1..10.

-- ─────────────────────────────────────────────────────────────────────────────
-- 1. Campaign lifecycle on assessment_periods (Section 5, FR-AS1, FR-AS13 size).
-- ─────────────────────────────────────────────────────────────────────────────
ALTER TABLE assessment_periods
    ADD COLUMN IF NOT EXISTS status       text NOT NULL DEFAULT 'draft',
    ADD COLUMN IF NOT EXISTS group_size   integer NOT NULL DEFAULT 12,
    ADD COLUMN IF NOT EXISTS confirmed_at timestamptz,
    ADD COLUMN IF NOT EXISTS published_at timestamptz;

ALTER TABLE assessment_periods DROP CONSTRAINT IF EXISTS assessment_periods_status_check;
ALTER TABLE assessment_periods ADD CONSTRAINT assessment_periods_status_check
    CHECK (status IN ('draft', 'assigned', 'in_progress', 'admin_review', 'confirmed', 'published'));

ALTER TABLE assessment_periods DROP CONSTRAINT IF EXISTS assessment_periods_group_size_check;
ALTER TABLE assessment_periods ADD CONSTRAINT assessment_periods_group_size_check
    CHECK (group_size >= 1);

-- A campaign may target several departments and several sections (FR-AS1).
-- The legacy single department_id stays as the "primary" department.
CREATE TABLE IF NOT EXISTS assessment_period_departments (
    period_id     uuid NOT NULL REFERENCES assessment_periods(id) ON DELETE CASCADE,
    department_id uuid NOT NULL REFERENCES departments(id),
    PRIMARY KEY (period_id, department_id)
);

CREATE TABLE IF NOT EXISTS assessment_period_sections (
    period_id  uuid NOT NULL REFERENCES assessment_periods(id) ON DELETE CASCADE,
    section_id uuid NOT NULL REFERENCES sections(id),
    PRIMARY KEY (period_id, section_id)
);

-- ─────────────────────────────────────────────────────────────────────────────
-- 2. Criteria (FR-AS3): the competencies selected for a campaign, each with an
--    optional name/description override and a minimum passing score (1..10).
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS assessment_criteria (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    period_id     uuid NOT NULL REFERENCES assessment_periods(id) ON DELETE CASCADE,
    competency_id uuid NOT NULL REFERENCES competencies(id),
    name          text,
    description   text,
    min_score     integer CHECK (min_score IS NULL OR (min_score >= 1 AND min_score <= 10)),
    sort_order    integer NOT NULL DEFAULT 0,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    UNIQUE (period_id, competency_id)
);

CREATE INDEX IF NOT EXISTS assessment_criteria_period_idx ON assessment_criteria (period_id);

CREATE TRIGGER assessment_criteria_set_updated_at BEFORE UPDATE ON assessment_criteria
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ─────────────────────────────────────────────────────────────────────────────
-- 3. Assessees (FR-AS2): the employees being evaluated in a campaign, with a
--    per-campaign status. assessment_participants keeps storing EVALUATORS.
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS assessment_assessees (
    id        uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    period_id uuid NOT NULL REFERENCES assessment_periods(id) ON DELETE CASCADE,
    user_id   uuid NOT NULL REFERENCES users(id),
    status    text NOT NULL DEFAULT 'participant',
    added_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (period_id, user_id)
);

CREATE INDEX IF NOT EXISTS assessment_assessees_period_idx ON assessment_assessees (period_id);
CREATE INDEX IF NOT EXISTS assessment_assessees_user_idx   ON assessment_assessees (user_id);

-- ─────────────────────────────────────────────────────────────────────────────
-- 4. Per-assessee assessors (FR-AS4): one or more assessors per assessee.
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS assessment_assessee_assessors (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    period_id        uuid NOT NULL REFERENCES assessment_periods(id) ON DELETE CASCADE,
    assessee_user_id uuid NOT NULL REFERENCES users(id),
    assessor_user_id uuid NOT NULL REFERENCES users(id),
    added_at         timestamptz NOT NULL DEFAULT now(),
    UNIQUE (period_id, assessee_user_id, assessor_user_id)
);

CREATE INDEX IF NOT EXISTS aaa_period_assessee_idx ON assessment_assessee_assessors (period_id, assessee_user_id);
CREATE INDEX IF NOT EXISTS aaa_period_assessor_idx ON assessment_assessee_assessors (period_id, assessor_user_id);

-- ─────────────────────────────────────────────────────────────────────────────
-- 5. Scoring: snapshot the auto interpretation + final user text on each score
--    (FR-AS7.2 rules 24, 25), and tighten the scale to 1..10 (Section 7 rule 1).
-- ─────────────────────────────────────────────────────────────────────────────
ALTER TABLE assessment_scores
    ADD COLUMN IF NOT EXISTS auto_interpretation text;
-- feedback already exists and now holds the FINAL (possibly edited) interpretation.

ALTER TABLE assessment_scores DROP CONSTRAINT IF EXISTS assessment_scores_score_check;
ALTER TABLE assessment_scores ADD CONSTRAINT assessment_scores_score_check
    CHECK (score IS NULL OR (score >= 1 AND score <= 10));

-- ─────────────────────────────────────────────────────────────────────────────
-- 6. Text-interpretation reference / справочник (FR-AS7.2.3, 7.2.4).
--    Active record keyed by (department, grade, competency, score). Versioned;
--    every change is also written to *_history.
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS assessment_interpretations (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    department_id uuid NOT NULL REFERENCES departments(id),
    grade_id      uuid NOT NULL REFERENCES grades(id),
    competency_id uuid NOT NULL REFERENCES competencies(id),
    score         integer NOT NULL CHECK (score >= 1 AND score <= 10),
    text          text NOT NULL,
    version       integer NOT NULL DEFAULT 1,
    is_active     boolean NOT NULL DEFAULT true,
    created_by    uuid REFERENCES users(id),
    updated_by    uuid REFERENCES users(id),
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS assessment_interpretations_key_uniq
    ON assessment_interpretations (department_id, grade_id, competency_id, score)
    WHERE is_active;

CREATE INDEX IF NOT EXISTS assessment_interpretations_lookup_idx
    ON assessment_interpretations (department_id, grade_id, competency_id, score);

CREATE TRIGGER assessment_interpretations_set_updated_at BEFORE UPDATE ON assessment_interpretations
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS assessment_interpretation_history (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    interpretation_id uuid REFERENCES assessment_interpretations(id) ON DELETE SET NULL,
    department_id     uuid NOT NULL,
    grade_id          uuid NOT NULL,
    competency_id     uuid NOT NULL,
    score             integer NOT NULL,
    text              text NOT NULL,
    version           integer NOT NULL,
    action            text NOT NULL CHECK (action IN ('create', 'update', 'delete', 'copy')),
    changed_by        uuid REFERENCES users(id),
    changed_at        timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS assessment_interpretation_history_iid_idx
    ON assessment_interpretation_history (interpretation_id, changed_at DESC);
CREATE INDEX IF NOT EXISTS assessment_interpretation_history_key_idx
    ON assessment_interpretation_history (department_id, grade_id, competency_id, score, changed_at DESC);

-- ─────────────────────────────────────────────────────────────────────────────
-- 7. Learning-group engine (FR-AS13).
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS learning_groups (
    id                     uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    period_id              uuid NOT NULL REFERENCES assessment_periods(id) ON DELETE CASCADE,
    group_no               integer NOT NULL,
    score_min              numeric(4,2),
    score_max              numeric(4,2),
    strength_competency_id uuid REFERENCES competencies(id),
    strength_score         numeric(4,2),
    confirmed              boolean NOT NULL DEFAULT false,
    formed_at              timestamptz NOT NULL DEFAULT now(),
    UNIQUE (period_id, group_no)
);

CREATE INDEX IF NOT EXISTS learning_groups_period_idx ON learning_groups (period_id);

CREATE TABLE IF NOT EXISTS learning_group_members (
    id        uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id  uuid NOT NULL REFERENCES learning_groups(id) ON DELETE CASCADE,
    period_id uuid NOT NULL REFERENCES assessment_periods(id) ON DELETE CASCADE,
    user_id   uuid NOT NULL REFERENCES users(id),
    avg_score numeric(4,2) NOT NULL,
    position  integer NOT NULL DEFAULT 0,
    UNIQUE (period_id, user_id)   -- one employee in only one group per campaign
);

CREATE INDEX IF NOT EXISTS learning_group_members_group_idx ON learning_group_members (group_id);

CREATE TABLE IF NOT EXISTS learning_group_dev_zones (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id      uuid NOT NULL REFERENCES learning_groups(id) ON DELETE CASCADE,
    competency_id uuid NOT NULL REFERENCES competencies(id),
    avg_score     numeric(4,2) NOT NULL,
    rank          integer NOT NULL,
    UNIQUE (group_id, competency_id)
);

CREATE INDEX IF NOT EXISTS learning_group_dev_zones_group_idx ON learning_group_dev_zones (group_id);

-- Journal of all (re)formations and manual edits to groups (FR-AS13.11, rule 13).
CREATE TABLE IF NOT EXISTS learning_group_journal (
    id        uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    period_id uuid NOT NULL REFERENCES assessment_periods(id) ON DELETE CASCADE,
    group_id  uuid REFERENCES learning_groups(id) ON DELETE SET NULL,
    action    text NOT NULL,
    detail    jsonb,
    actor_id  uuid REFERENCES users(id),
    at        timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS learning_group_journal_period_idx ON learning_group_journal (period_id, at DESC);

-- ─────────────────────────────────────────────────────────────────────────────
-- 8. Talent Profile sync fields (FR-AS11).
-- ─────────────────────────────────────────────────────────────────────────────
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS last_assessment_at     timestamptz,
    ADD COLUMN IF NOT EXISTS last_assessment_status text;
