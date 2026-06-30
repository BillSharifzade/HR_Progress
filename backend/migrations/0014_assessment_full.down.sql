-- Reverse 0014.

ALTER TABLE users
    DROP COLUMN IF EXISTS last_assessment_at,
    DROP COLUMN IF EXISTS last_assessment_status;

DROP TABLE IF EXISTS learning_group_journal;
DROP TABLE IF EXISTS learning_group_dev_zones;
DROP TABLE IF EXISTS learning_group_members;
DROP TABLE IF EXISTS learning_groups;

DROP TABLE IF EXISTS assessment_interpretation_history;
DROP TABLE IF EXISTS assessment_interpretations;

ALTER TABLE assessment_scores DROP CONSTRAINT IF EXISTS assessment_scores_score_check;
ALTER TABLE assessment_scores ADD CONSTRAINT assessment_scores_score_check
    CHECK (score >= 0 AND score <= 10);
ALTER TABLE assessment_scores DROP COLUMN IF EXISTS auto_interpretation;

DROP TABLE IF EXISTS assessment_assessee_assessors;
DROP TABLE IF EXISTS assessment_assessees;
DROP TABLE IF EXISTS assessment_criteria;
DROP TABLE IF EXISTS assessment_period_sections;
DROP TABLE IF EXISTS assessment_period_departments;

ALTER TABLE assessment_periods DROP CONSTRAINT IF EXISTS assessment_periods_status_check;
ALTER TABLE assessment_periods DROP CONSTRAINT IF EXISTS assessment_periods_group_size_check;
ALTER TABLE assessment_periods
    DROP COLUMN IF EXISTS status,
    DROP COLUMN IF EXISTS group_size,
    DROP COLUMN IF EXISTS confirmed_at,
    DROP COLUMN IF EXISTS published_at;
