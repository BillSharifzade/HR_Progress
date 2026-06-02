ALTER TABLE sections ADD COLUMN IF NOT EXISTS code text;

CREATE UNIQUE INDEX sections_code_dept_uniq
    ON sections (department_id, lower(code))
    WHERE deleted_at IS NULL AND code IS NOT NULL;
