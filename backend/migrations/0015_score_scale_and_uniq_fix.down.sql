-- Reverse 0015: narrow the mark scale back to 1..10.
--
-- The legacy UNIQUE (period_id, employee_id, competency_id, assessor_role) is
-- deliberately NOT restored: 0012 already intended to drop it and the current
-- scoring model (an ASSESSOR group where several users score the same cell)
-- cannot satisfy it.
ALTER TABLE assessment_interpretations
    DROP CONSTRAINT IF EXISTS assessment_interpretations_score_check;
ALTER TABLE assessment_interpretations ADD CONSTRAINT assessment_interpretations_score_check
    CHECK (score >= 1 AND score <= 10);

ALTER TABLE assessment_scores DROP CONSTRAINT IF EXISTS assessment_scores_score_check;
ALTER TABLE assessment_scores ADD CONSTRAINT assessment_scores_score_check
    CHECK (score IS NULL OR (score >= 1 AND score <= 10));
