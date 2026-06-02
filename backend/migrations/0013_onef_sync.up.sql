-- 1F (Первая форма) sync integration.
--
-- Adds:
--   - users.employee_no: local auto-incrementing employee ID (sequence).
--   - users.one_f_user_id: 1F's external UserID (unique when present).
--   - users.one_f_manager_user_id: 1F UserID of the worker's supervisor (raw, unresolved).
--   - users.one_f_is_manager: cached IsManager flag from 1F.
--   - users.phone_number: from 1F PhoneNumber.
--   - users.last_synced_at: last successful 1F sync timestamp for this user.
-- Plus the onef_sync_runs audit table for poll history.

-- Sequence for auto-incrementing employee numbers.
CREATE SEQUENCE IF NOT EXISTS users_employee_no_seq START WITH 1 INCREMENT BY 1;

ALTER TABLE users
    ADD COLUMN employee_no            bigint,
    ADD COLUMN one_f_user_id          bigint,
    ADD COLUMN one_f_manager_user_id  bigint,
    ADD COLUMN one_f_is_manager       boolean NOT NULL DEFAULT false,
    ADD COLUMN phone_number           text,
    ADD COLUMN last_synced_at         timestamptz;

-- Backfill employee_no for existing rows in deterministic order.
UPDATE users SET employee_no = nextval('users_employee_no_seq')
WHERE employee_no IS NULL;

ALTER TABLE users
    ALTER COLUMN employee_no SET NOT NULL,
    ALTER COLUMN employee_no SET DEFAULT nextval('users_employee_no_seq');

ALTER SEQUENCE users_employee_no_seq OWNED BY users.employee_no;

CREATE UNIQUE INDEX users_employee_no_uniq
    ON users (employee_no)
    WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX users_one_f_user_id_uniq
    ON users (one_f_user_id)
    WHERE deleted_at IS NULL AND one_f_user_id IS NOT NULL;

CREATE INDEX users_one_f_manager_user_id_idx
    ON users (one_f_manager_user_id)
    WHERE one_f_manager_user_id IS NOT NULL;

-- Sync-run audit log: one row per poll attempt, successful or not.
CREATE TABLE onef_sync_runs (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    started_at      timestamptz NOT NULL DEFAULT now(),
    finished_at     timestamptz,
    triggered_by    uuid REFERENCES users(id),
    trigger_kind    text NOT NULL,        -- 'cron' | 'manual'
    status          text NOT NULL,        -- 'running' | 'success' | 'failed'
    fetched_count   integer NOT NULL DEFAULT 0,
    created_count   integer NOT NULL DEFAULT 0,
    updated_count   integer NOT NULL DEFAULT 0,
    skipped_count   integer NOT NULL DEFAULT 0,
    error_message   text,
    duration_ms     integer
);

CREATE INDEX onef_sync_runs_started_at_idx ON onef_sync_runs (started_at DESC);
