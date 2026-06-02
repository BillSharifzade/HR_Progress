DROP TABLE IF EXISTS onef_sync_runs;

DROP INDEX IF EXISTS users_one_f_manager_user_id_idx;
DROP INDEX IF EXISTS users_one_f_user_id_uniq;
DROP INDEX IF EXISTS users_employee_no_uniq;

ALTER TABLE users
    DROP COLUMN IF EXISTS last_synced_at,
    DROP COLUMN IF EXISTS phone_number,
    DROP COLUMN IF EXISTS one_f_is_manager,
    DROP COLUMN IF EXISTS one_f_manager_user_id,
    DROP COLUMN IF EXISTS one_f_user_id,
    DROP COLUMN IF EXISTS employee_no;

DROP SEQUENCE IF EXISTS users_employee_no_seq;
