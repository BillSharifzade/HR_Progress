-- 1. Drop the legacy UNIQUE (period_id, employee_id, competency_id, assessor_role)
--    on assessment_scores — for real this time.
--
--    0012 already intended to drop it (the ASSESSOR group means several users
--    legitimately write rows with the same assessor_role for one cell), but it
--    spelled the constraint name `..._competency_id_assess_key` while Postgres
--    had truncated the generated name to `..._competency_id_asses_key` (63-byte
--    limit). DROP ... IF EXISTS silently matched nothing, so the constraint
--    survived: the first assessor to score a cell inserted fine and every
--    other assigned assessor got a 23505 → 500 → "Не удалось сохранить".
--
--    Look the constraint up by its column set instead of by name so the
--    truncation trap cannot bite again.
DO $$
DECLARE
    con record;
BEGIN
    FOR con IN
        SELECT c.conname
          FROM pg_constraint c
          JOIN pg_class t ON t.oid = c.conrelid
          JOIN pg_namespace n ON n.oid = t.relnamespace
         WHERE n.nspname = current_schema()
           AND t.relname = 'assessment_scores'
           AND c.contype = 'u'
           AND (
                 SELECT array_agg(a.attname::text ORDER BY a.attname)
                   FROM unnest(c.conkey) AS k(attnum)
                   JOIN pg_attribute a
                     ON a.attrelid = c.conrelid AND a.attnum = k.attnum
               ) = ARRAY['assessor_role', 'competency_id', 'employee_id', 'period_id']
    LOOP
        EXECUTE format('ALTER TABLE assessment_scores DROP CONSTRAINT %I', con.conname);
    END LOOP;
END $$;

-- 2. Widen the mark scale from 1..10 to 0..10. A 0 is a meaningful mark
--    ("competency not demonstrated at all"), and 0014 had narrowed the scale
--    so that saving one was rejected outright.
ALTER TABLE assessment_scores DROP CONSTRAINT IF EXISTS assessment_scores_score_check;
ALTER TABLE assessment_scores ADD CONSTRAINT assessment_scores_score_check
    CHECK (score IS NULL OR (score >= 0 AND score <= 10));

-- The interpretation reference is keyed by score, so it has to cover 0 too —
-- otherwise a 0 mark could never get an auto-interpretation text.
ALTER TABLE assessment_interpretations
    DROP CONSTRAINT IF EXISTS assessment_interpretations_score_check;
ALTER TABLE assessment_interpretations ADD CONSTRAINT assessment_interpretations_score_check
    CHECK (score >= 0 AND score <= 10);
