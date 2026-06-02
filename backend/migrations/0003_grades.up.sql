-- Global grade ladder.

CREATE TABLE grades (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name        text NOT NULL,
    level       integer NOT NULL,
    description text,
    is_active   boolean NOT NULL DEFAULT true,
    deleted_at  timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX grades_name_uniq ON grades (name) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX grades_level_uniq ON grades (level) WHERE deleted_at IS NULL;

CREATE TRIGGER grades_set_updated_at BEFORE UPDATE ON grades
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
