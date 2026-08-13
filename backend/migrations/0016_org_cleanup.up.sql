-- 0016: org cleanup — merge the mis-named department shells with the rows that
-- actually hold the people, then give the survivors their real names.
--
-- Background. Migration 0005 expanded two bare abbreviations from the source
-- xlsx by guess, and guessed both wrong:
--
--   `ДФП` was seeded as "Департамент Финансового Планирования" — no such
--         department exists in 1F. The real ДФП is "Департамент
--         Фармацевтической Промоции".
--   `БЮД` was seeded as "Бюджетный Департамент" — likewise absent from 1F.
--         The real one is "Бухгалтерский и Юридический Департамент".
--
-- The consequence: each competency matrix (51 requirements apiece) sits on a
-- department with zero staff, while the department that has the staff has no
-- matrix. On production the staff-holding rows arrived via the 1F sync under
-- de-duplicated codes — `ДФП2` (auto-created when the old shell still held the
-- `ДФП` code) and `БИЮД`.
--
-- This migration is deliberately CONDITIONAL and IDEMPOTENT. Local and
-- production have diverged, and part of the cleanup was already applied to
-- production by hand, so every block must be a no-op when its precondition is
-- absent and safe to run a second time. On a database seeded fresh from 0005
-- there is no `ДФП2`/`БИЮД` at all: the merge loop skips, and only the two
-- renames apply — which is exactly right for a new install.
--
-- Policy on collisions: duplicates that carry no information (a repeated role
-- grant, a repeated period↔department link) are dropped; anything holding real
-- content (requirements, interpretations, sections) raises instead of being
-- silently discarded. The migration should fail loudly rather than lose data.

DO $$
DECLARE
    pair    record;
    src_id  uuid;
    dst_id  uuid;
    n       int;
    leftover int;
BEGIN
    FOR pair IN
        SELECT * FROM (VALUES
            ('ДФП2', 'ДФП'),
            ('БИЮД', 'БЮД')
        ) AS t(src_code, dst_code)
    LOOP
        -- Resolve both sides. `departments_code_uniq` is partial
        -- (WHERE deleted_at IS NULL), so a code can legitimately repeat across
        -- soft-deleted rows — resolve by code without filtering on deleted_at
        -- (both sides may be soft-deleted) but insist the answer is unambiguous.
        SELECT count(*) INTO n FROM departments WHERE code = pair.src_code;
        IF n = 0 THEN
            RAISE NOTICE '0016: no department %, nothing to merge', pair.src_code;
            CONTINUE;
        ELSIF n > 1 THEN
            RAISE EXCEPTION '0016: % matches % department rows, cannot merge unambiguously',
                pair.src_code, n;
        END IF;

        SELECT count(*) INTO n FROM departments WHERE code = pair.dst_code;
        IF n = 0 THEN
            RAISE NOTICE '0016: no target department %, skipping merge of %',
                pair.dst_code, pair.src_code;
            CONTINUE;
        ELSIF n > 1 THEN
            RAISE EXCEPTION '0016: % matches % department rows, cannot merge unambiguously',
                pair.dst_code, n;
        END IF;

        SELECT id INTO src_id FROM departments WHERE code = pair.src_code;
        SELECT id INTO dst_id FROM departments WHERE code = pair.dst_code;

        RAISE NOTICE '0016: merging % (%) into % (%)',
            pair.src_code, src_id, pair.dst_code, dst_id;

        ------------------------------------------------------------------
        -- Guards: refuse to merge if it would destroy content.
        ------------------------------------------------------------------

        -- sections_code_dept_uniq (department_id, lower(code)) WHERE deleted_at IS NULL AND code IS NOT NULL
        SELECT count(*) INTO n
          FROM sections s_src
          JOIN sections s_dst
            ON s_dst.department_id = dst_id
           AND s_dst.deleted_at IS NULL
           AND s_dst.code IS NOT NULL
           AND lower(s_dst.code) = lower(s_src.code)
         WHERE s_src.department_id = src_id
           AND s_src.deleted_at IS NULL
           AND s_src.code IS NOT NULL;
        IF n > 0 THEN
            RAISE EXCEPTION '0016: % section code(s) collide moving % → %',
                n, pair.src_code, pair.dst_code;
        END IF;

        -- sections_dept_name_uniq (department_id, name) WHERE deleted_at IS NULL
        SELECT count(*) INTO n
          FROM sections s_src
          JOIN sections s_dst
            ON s_dst.department_id = dst_id
           AND s_dst.deleted_at IS NULL
           AND s_dst.name = s_src.name
         WHERE s_src.department_id = src_id
           AND s_src.deleted_at IS NULL;
        IF n > 0 THEN
            RAISE EXCEPTION '0016: % section name(s) collide moving % → %',
                n, pair.src_code, pair.dst_code;
        END IF;

        -- dept_competency_requirements (department_id, competency_id, grade_id)
        SELECT count(*) INTO n
          FROM dept_competency_requirements r_src
          JOIN dept_competency_requirements r_dst
            ON r_dst.department_id = dst_id
           AND r_dst.competency_id = r_src.competency_id
           AND r_dst.grade_id      = r_src.grade_id
         WHERE r_src.department_id = src_id;
        IF n > 0 THEN
            RAISE EXCEPTION '0016: % competency requirement(s) collide moving % → %',
                n, pair.src_code, pair.dst_code;
        END IF;

        -- assessment_interpretations (department_id, grade_id, competency_id, score) WHERE is_active
        SELECT count(*) INTO n
          FROM assessment_interpretations i_src
          JOIN assessment_interpretations i_dst
            ON i_dst.department_id = dst_id
           AND i_dst.grade_id      = i_src.grade_id
           AND i_dst.competency_id = i_src.competency_id
           AND i_dst.score         = i_src.score
           AND i_dst.is_active
         WHERE i_src.department_id = src_id
           AND i_src.is_active;
        IF n > 0 THEN
            RAISE EXCEPTION '0016: % interpretation(s) collide moving % → %',
                n, pair.src_code, pair.dst_code;
        END IF;

        ------------------------------------------------------------------
        -- Drop information-free duplicates, then repoint.
        ------------------------------------------------------------------

        -- user_roles_uniq is (user_id, role, COALESCE(dept,0…0), COALESCE(section,0…0)).
        -- Someone holding the same role scoped to both source and target would
        -- collide on repoint; the grant is identical either way, so drop one.
        DELETE FROM user_roles ur_src
         WHERE ur_src.scope_department_id = src_id
           AND EXISTS (
               SELECT 1 FROM user_roles ur_dst
                WHERE ur_dst.user_id = ur_src.user_id
                  AND ur_dst.role    = ur_src.role
                  AND ur_dst.scope_department_id = dst_id
                  AND COALESCE(ur_dst.scope_section_id, '00000000-0000-0000-0000-000000000000'::uuid)
                    = COALESCE(ur_src.scope_section_id, '00000000-0000-0000-0000-000000000000'::uuid)
           );

        -- assessment_period_departments is a pure link table (period_id, department_id).
        DELETE FROM assessment_period_departments apd_src
         WHERE apd_src.department_id = src_id
           AND EXISTS (
               SELECT 1 FROM assessment_period_departments apd_dst
                WHERE apd_dst.period_id = apd_src.period_id
                  AND apd_dst.department_id = dst_id
           );

        -- All eight FK columns that reference departments, as of migration 15.
        -- (Verified against pg_constraint on production — an earlier audit run
        --  against a migration-13 database missed the last two.)
        UPDATE users                         SET department_id       = dst_id WHERE department_id       = src_id;
        UPDATE sections                      SET department_id       = dst_id WHERE department_id       = src_id;
        UPDATE user_roles                    SET scope_department_id = dst_id WHERE scope_department_id = src_id;
        UPDATE audit_logs                    SET department_scope_id = dst_id WHERE department_scope_id = src_id;
        UPDATE dept_competency_requirements  SET department_id       = dst_id WHERE department_id       = src_id;
        UPDATE assessment_periods            SET department_id       = dst_id WHERE department_id       = src_id;
        UPDATE assessment_period_departments SET department_id       = dst_id WHERE department_id       = src_id;
        UPDATE assessment_interpretations    SET department_id       = dst_id WHERE department_id       = src_id;

        ------------------------------------------------------------------
        -- Only now remove the source, and only if truly unreferenced.
        ------------------------------------------------------------------
        SELECT (SELECT count(*) FROM users                         WHERE department_id       = src_id)
             + (SELECT count(*) FROM sections                      WHERE department_id       = src_id)
             + (SELECT count(*) FROM user_roles                    WHERE scope_department_id = src_id)
             + (SELECT count(*) FROM audit_logs                    WHERE department_scope_id = src_id)
             + (SELECT count(*) FROM dept_competency_requirements  WHERE department_id       = src_id)
             + (SELECT count(*) FROM assessment_periods            WHERE department_id       = src_id)
             + (SELECT count(*) FROM assessment_period_departments WHERE department_id       = src_id)
             + (SELECT count(*) FROM assessment_interpretations    WHERE department_id       = src_id)
          INTO leftover;

        IF leftover <> 0 THEN
            RAISE EXCEPTION '0016: % still has % referencing row(s) after repoint, refusing to delete',
                pair.src_code, leftover;
        END IF;

        DELETE FROM departments WHERE id = src_id;
    END LOOP;
END $$;

-- Give the survivors their real names, and revive them if a well-meaning
-- cleanup soft-deleted the shell that holds the matrix.
--
-- This must run AFTER the merge loop: `departments_name_uniq` is partial on
-- (name) WHERE deleted_at IS NULL, and until `ДФП2` is gone it still occupies
-- "Департамент Фармацевтической Промоции".
UPDATE departments
   SET name       = 'Департамент Фармацевтической Промоции',
       deleted_at = NULL,
       is_active  = true
 WHERE code = 'ДФП'
   AND (name <> 'Департамент Фармацевтической Промоции' OR deleted_at IS NOT NULL OR NOT is_active);

UPDATE departments
   SET name       = 'Бухгалтерский и Юридический Департамент',
       deleted_at = NULL,
       is_active  = true
 WHERE code = 'БЮД'
   AND (name <> 'Бухгалтерский и Юридический Департамент' OR deleted_at IS NOT NULL OR NOT is_active);
