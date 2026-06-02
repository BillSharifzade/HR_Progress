DROP TABLE IF EXISTS user_certifications;

ALTER TABLE users
    DROP COLUMN IF EXISTS hobbies,
    DROP COLUMN IF EXISTS position_id,
    DROP COLUMN IF EXISTS birth_date,
    DROP COLUMN IF EXISTS personnel_number;

DROP TABLE IF EXISTS positions;

-- Enum values cannot be removed in PostgreSQL; role_kind additions are permanent.
