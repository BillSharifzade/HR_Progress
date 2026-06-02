-- Reshape assessment scoring around per-user participants.
-- Each assessment_period now has an explicit, locked list of participants
-- (the 4 individual evaluator slots + the ASSESSOR group). Scoring rows
-- are now bound to a user_id, not just a role string.

-- 1. Extend the assessor_role CHECK so 'ASSESSOR' (group) is valid alongside
--    the four individual slots.
ALTER TABLE assessment_scores DROP CONSTRAINT IF EXISTS assessment_scores_assessor_role_check;
ALTER TABLE assessment_scores ADD CONSTRAINT assessment_scores_assessor_role_check
    CHECK (assessor_role IN ('HEAD', 'DEPT_HEAD', 'HRA', 'DCR_HEAD', 'ASSESSOR'));

-- 2. Bind every score to the user who wrote it. Nullable for legacy rows.
ALTER TABLE assessment_scores ADD COLUMN IF NOT EXISTS user_id uuid REFERENCES users(id);

-- Drop the legacy unique on (period, employee, competency, assessor_role) —
-- with the ASSESSOR group several users will write rows with the same role.
ALTER TABLE assessment_scores DROP CONSTRAINT IF EXISTS assessment_scores_period_id_employee_id_competency_id_assess_key;

-- New uniqueness: one score per (period, worker, competency, user).
CREATE UNIQUE INDEX IF NOT EXISTS assessment_scores_pwcu_uniq
    ON assessment_scores (period_id, employee_id, competency_id, user_id)
    WHERE user_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS assessment_scores_user_idx ON assessment_scores (user_id);

-- 3. Participants table — the people assigned to score in a given period.
CREATE TABLE assessment_participants (
    id        uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    period_id uuid NOT NULL REFERENCES assessment_periods(id) ON DELETE CASCADE,
    user_id   uuid NOT NULL REFERENCES users(id),
    role      text NOT NULL CHECK (role IN ('HEAD', 'DEPT_HEAD', 'HRA', 'DCR_HEAD', 'ASSESSOR')),
    added_at  timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX assessment_participants_uniq
    ON assessment_participants (period_id, user_id, role);
CREATE INDEX assessment_participants_period_idx ON assessment_participants (period_id);
CREATE INDEX assessment_participants_user_idx   ON assessment_participants (user_id);

-- 4. Consolidated mark per (period, worker, competency).
-- Filled by the backend once every ASSESSOR-group participant has scored
-- the (worker, competency) cell. avg_score is the simple mean of those
-- assessor scores.
CREATE TABLE assessment_consolidated (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    period_id     uuid NOT NULL REFERENCES assessment_periods(id) ON DELETE CASCADE,
    employee_id   uuid NOT NULL REFERENCES users(id),
    competency_id uuid NOT NULL REFERENCES competencies(id),
    avg_score     numeric(4,2) NOT NULL,
    finalized_at  timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX assessment_consolidated_uniq
    ON assessment_consolidated (period_id, employee_id, competency_id);
CREATE INDEX assessment_consolidated_period_idx ON assessment_consolidated (period_id);
