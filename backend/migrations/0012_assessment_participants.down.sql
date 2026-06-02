DROP TABLE IF EXISTS assessment_consolidated;
DROP TABLE IF EXISTS assessment_participants;

DROP INDEX IF EXISTS assessment_scores_user_idx;
DROP INDEX IF EXISTS assessment_scores_pwcu_uniq;

ALTER TABLE assessment_scores DROP COLUMN IF EXISTS user_id;

ALTER TABLE assessment_scores DROP CONSTRAINT IF EXISTS assessment_scores_assessor_role_check;
ALTER TABLE assessment_scores ADD CONSTRAINT assessment_scores_assessor_role_check
    CHECK (assessor_role IN ('HEAD', 'DEPT_HEAD', 'HRA', 'DCR_HEAD'));

ALTER TABLE assessment_scores ADD CONSTRAINT assessment_scores_period_id_employee_id_competency_id_assess_key
    UNIQUE (period_id, employee_id, competency_id, assessor_role);
